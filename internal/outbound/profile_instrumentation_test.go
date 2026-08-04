package outbound_test

import (
	"bytes"
	"context"
	"encoding/json"
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
}

type staticProfileClient struct {
	profile profile.Profile
	err     error
}

func (c staticProfileClient) FetchProfile(context.Context, int64, string) (profile.Profile, error) {
	return c.profile, c.err
}
