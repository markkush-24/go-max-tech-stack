package middleware_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"pet-study/internal/middleware"
	"pet-study/internal/requestid"
	"strings"
	"testing"
)

func TestRecover_LogsStackTrace_AndReturnsGenericProblem(t *testing.T) {
	var logBuf bytes.Buffer

	prev := slog.Default()
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{}))
	slog.SetDefault(logger)
	t.Cleanup(func() {
		slog.SetDefault(prev)
	})

	h := middleware.Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/panic", nil)
	req = req.WithContext(requestid.WithRequestID(req.Context(), "rid-123"))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d want=%d", resp.StatusCode, http.StatusInternalServerError)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	bodyText := string(body)
	if !strings.Contains(bodyText, `"detail":"internal server error"`) {
		t.Fatalf("body=%q must contain generic error detail", bodyText)
	}
	if strings.Contains(bodyText, "goroutine") || strings.Contains(bodyText, "recover_test.go") {
		t.Fatalf("body=%q must not leak stack trace details", bodyText)
	}

	logText := logBuf.String()
	if !strings.Contains(logText, "panic recovered") {
		t.Fatalf("logs=%q must contain recovery message", logText)
	}
	if !strings.Contains(logText, "request_id=rid-123") {
		t.Fatalf("logs=%q must contain request id", logText)
	}
	if !strings.Contains(logText, "stack=") {
		t.Fatalf("logs=%q must contain stack field", logText)
	}
	if !strings.Contains(logText, "goroutine") {
		t.Fatalf("logs=%q must contain stack trace content", logText)
	}
}
