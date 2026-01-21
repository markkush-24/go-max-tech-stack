package router_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"pet-study/internal/health"
	"pet-study/internal/requestid"
	"pet-study/internal/router"
	"testing"
	"time"
)

type healthResp struct {
	Status  string            `json:"status"`
	TS      string            `json:"ts"`
	Details map[string]string `json:"details,omitempty"`
}

func decodeHealth(t *testing.T, rec *httptest.ResponseRecorder) healthResp {
	t.Helper()
	var out healthResp
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode health json: %v body=%q", err, rec.Body.String())
	}
	return out
}

func mustParseRFC3339Nano(t *testing.T, s string) {
	t.Helper()
	if s == "" {
		t.Fatalf("ts is empty")
	}
	if _, err := time.Parse(time.RFC3339Nano, s); err != nil {
		t.Fatalf("ts=%q is not RFC3339Nano: %v", s, err)
	}
}

func newRootForHealth(readiness *health.Readiness) http.Handler {
	healthRouter := router.NewHealthRouter(readiness)

	// app в этих тестах не нужен, но root требует handler.
	app := http.NewServeMux()
	app.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	root := router.NewRoot(app, healthRouter, nil)
	return requestid.RequestIDMiddleware(root)
}

func TestLivez_Always200(t *testing.T) {
	h := newRootForHealth(health.NewReadiness())

	req := httptest.NewRequest(http.MethodGet, "http://example.com/livez", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want=200 body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control=%q want=no-store", got)
	}
	if ct := rec.Header().Get("Content-Type"); ct == "" {
		t.Fatalf("Content-Type empty")
	}
	if rid := rec.Header().Get(requestid.HeaderName); rid == "" {
		t.Fatalf("%s empty", requestid.HeaderName)
	}

	body := decodeHealth(t, rec)
	if body.Status != "ok" {
		t.Fatalf("body.status=%q want=ok", body.Status)
	}
	mustParseRFC3339Nano(t, body.TS)
}

func TestReadyz_NotReady_FailFast503(t *testing.T) {
	// default readiness = false
	h := newRootForHealth(health.NewReadiness(health.Check{
		Name: "repo",
		Fn:   func(ctx context.Context) error { return nil },
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/readyz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want=503 body=%s", rec.Code, rec.Body.String())
	}
	body := decodeHealth(t, rec)
	if body.Status != "not_ready" {
		t.Fatalf("body.status=%q want=not_ready", body.Status)
	}
	if body.Details == nil || body.Details["lifecycle"] == "" {
		t.Fatalf("expected details.lifecycle, got=%v", body.Details)
	}
}

func TestReadyz_Ready_AllChecksOK_200(t *testing.T) {
	rd := health.NewReadiness(
		health.Check{Name: "repo", Fn: func(ctx context.Context) error { return nil }},
	)
	rd.SetReady()
	h := newRootForHealth(rd)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/readyz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want=200 body=%s", rec.Code, rec.Body.String())
	}
	body := decodeHealth(t, rec)
	if body.Status != "ok" {
		t.Fatalf("body.status=%q want=ok", body.Status)
	}
	if body.Details != nil {
		t.Fatalf("expected no details, got=%v", body.Details)
	}
	mustParseRFC3339Nano(t, body.TS)
}

func TestReadyz_Ready_CheckFails_503WithDetails(t *testing.T) {
	rd := health.NewReadiness(
		health.Check{Name: "repo", Fn: func(ctx context.Context) error { return errors.New("boom") }},
	)
	rd.SetReady()
	h := newRootForHealth(rd)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/readyz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want=503 body=%s", rec.Code, rec.Body.String())
	}
	body := decodeHealth(t, rec)
	if body.Status != "not_ready" {
		t.Fatalf("body.status=%q want=not_ready", body.Status)
	}
	if body.Details == nil || body.Details["repo"] == "" {
		t.Fatalf("expected details.repo, got=%v", body.Details)
	}
}
