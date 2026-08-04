package outbound_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"testing"

	"pet-study/internal/config"
	"pet-study/internal/outbound"
	"pet-study/internal/outbound/profile"
)

func TestInstrumentedProfileClient_UsesCanonicalLogFieldsAndSingleComponent(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil)).
		With(config.LogFieldComponent, config.LogComponentOutboundProfile)

	baseURL, err := url.Parse("https://profile.example.test")
	if err != nil {
		t.Fatal(err)
	}

	client := outbound.NewInstrumentedProfileClient(baseURL, staticProfileClient{
		profile: profile.Profile{UserID: 1, Bio: "ok", City: "NY"},
	}, logger)

	_, err = client.FetchProfile(context.Background(), 1, "rid-1")
	if err != nil {
		t.Fatalf("FetchProfile() error = %v", err)
	}

	line := strings.TrimSpace(logs.String())
	if got := strings.Count(line, `"`+config.LogFieldComponent+`"`); got != 1 {
		t.Fatalf("log=%q component field count=%d want 1", line, got)
	}
	if strings.Contains(line, `"latency_ms"`) {
		t.Fatalf("log=%q must not include legacy latency_ms field", line)
	}

	var record map[string]any
	if err := json.Unmarshal([]byte(line), &record); err != nil {
		t.Fatalf("unmarshal log line: %v line=%q", err, line)
	}

	if record[config.LogFieldComponent] != config.LogComponentOutboundProfile {
		t.Fatalf("component=%v want %q", record[config.LogFieldComponent], config.LogComponentOutboundProfile)
	}
	if record[config.LogFieldMethod] != "GET" {
		t.Fatalf("method=%v want GET", record[config.LogFieldMethod])
	}
	if record[config.LogFieldRoute] != "/profiles/{user_id}" {
		t.Fatalf("route=%v want /profiles/{user_id}", record[config.LogFieldRoute])
	}
	if record[config.LogFieldStatus] != float64(200) {
		t.Fatalf("status=%v want 200", record[config.LogFieldStatus])
	}
	if _, ok := record[config.LogFieldDurationMS]; !ok {
		t.Fatalf("duration_ms missing from log record: %#v", record)
	}
	if _, ok := record["user_id"]; ok {
		t.Fatalf("record=%#v must not include user_id", record)
	}
}

func TestInstrumentedProfileClient_LogsErrorKindWithoutRawErrorOrUserID(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil)).
		With(config.LogFieldComponent, config.LogComponentOutboundProfile)

	baseURL, err := url.Parse("https://profile.example.test")
	if err != nil {
		t.Fatal(err)
	}

	client := outbound.NewInstrumentedProfileClient(baseURL, staticProfileClient{
		err: &profile.Error{
			Kind:  profile.ErrNetwork,
			Cause: errors.New("dial https://secret.example.test/profile?token=secret-token"),
		},
	}, logger)

	_, err = client.FetchProfile(context.Background(), 42, "rid-1")
	if err == nil {
		t.Fatal("FetchProfile() error = nil, want error")
	}

	line := strings.TrimSpace(logs.String())
	if strings.Contains(line, "secret.example.test") || strings.Contains(line, "secret-token") {
		t.Fatalf("log=%q must not include raw upstream error details", line)
	}

	record := decodeOutboundLogRecord(t, line)
	if record[logFieldErrorKindForTest] != "network" {
		t.Fatalf("error_kind=%v want network", record[logFieldErrorKindForTest])
	}
	if _, ok := record[config.LogFieldError]; ok {
		t.Fatalf("record=%#v must not include raw err field", record)
	}
	if _, ok := record["user_id"]; ok {
		t.Fatalf("record=%#v must not include user_id", record)
	}
}

func TestRetryingProfileClient_LogsGroupedRetryOperation(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil)).
		With(config.LogFieldComponent, config.LogComponentOutboundProfile)

	client := outbound.NewRetryingProfileClientWithLogger(3, 0, 0, &sequenceProfileClient{
		errs: []error{
			&profile.Error{Kind: profile.ErrNetwork, Cause: errors.New("secret network detail")},
			&profile.Error{Kind: profile.ErrUpstream5xx, Status: 503},
			nil,
		},
		profile: profile.Profile{UserID: 42, Bio: "ok", City: "NY"},
	}, logger)

	_, err := client.FetchProfile(context.Background(), 42, "rid-retry")
	if err != nil {
		t.Fatalf("FetchProfile() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("log lines=%q want 3 retry operation records", logs.String())
	}

	records := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, "secret network detail") {
			t.Fatalf("log=%q must not include raw error details", line)
		}
		record := decodeOutboundLogRecord(t, line)
		if record[config.LogFieldRequestID] != "rid-retry" {
			t.Fatalf("request_id=%v want rid-retry", record[config.LogFieldRequestID])
		}
		if record[logFieldOperationForTest] != "profile.fetch" {
			t.Fatalf("operation=%v want profile.fetch", record[logFieldOperationForTest])
		}
		records = append(records, record)
	}

	if records[0][logFieldEventForTest] != "profile.retry.scheduled" ||
		records[1][logFieldEventForTest] != "profile.retry.scheduled" ||
		records[2][logFieldEventForTest] != "profile.retry.completed" {
		t.Fatalf("events=%v, %v, %v", records[0][logFieldEventForTest], records[1][logFieldEventForTest], records[2][logFieldEventForTest])
	}
	if records[0]["attempt"] != float64(1) || records[1]["attempt"] != float64(2) {
		t.Fatalf("attempts=%v,%v want 1,2", records[0]["attempt"], records[1]["attempt"])
	}
	if records[2]["outcome"] != "success" || records[2]["attempts"] != float64(3) {
		t.Fatalf("final retry record=%#v want success after 3 attempts", records[2])
	}
}

type staticProfileClient struct {
	profile profile.Profile
	err     error
}

func (c staticProfileClient) FetchProfile(context.Context, int64, string) (profile.Profile, error) {
	return c.profile, c.err
}

type sequenceProfileClient struct {
	errs    []error
	profile profile.Profile
	calls   int
}

func (c *sequenceProfileClient) FetchProfile(context.Context, int64, string) (profile.Profile, error) {
	err := c.errs[c.calls]
	c.calls++
	if err != nil {
		return profile.Profile{}, err
	}
	return c.profile, nil
}

func decodeOutboundLogRecord(t *testing.T, line string) map[string]any {
	t.Helper()

	var record map[string]any
	if err := json.Unmarshal([]byte(line), &record); err != nil {
		t.Fatalf("unmarshal log line: %v line=%q", err, line)
	}
	return record
}

const (
	logFieldEventForTest     = "event"
	logFieldOperationForTest = "operation"
	logFieldErrorKindForTest = "error_kind"
)
