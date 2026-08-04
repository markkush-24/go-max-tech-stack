package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pet-study/internal/config"
	"pet-study/internal/httputils"
	"pet-study/internal/requestid"
	"pet-study/internal/security"
)

func TestAuthenticate_LogsSafeDeniedReasonWithoutToken(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil)).
		With(config.LogFieldComponent, logComponentHTTPSecurity)

	auth, err := NewAuthAPIWithLogger(failingVerifier{}, logger)
	if err != nil {
		t.Fatalf("NewAuthAPIWithLogger: %v", err)
	}

	next := httputils.AppHandler(func(http.ResponseWriter, *http.Request) error {
		t.Fatal("next handler was called")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Pattern = "GET /api/v1/users"
	req = req.WithContext(requestid.WithRequestID(req.Context(), "rid-authn"))
	rec := httptest.NewRecorder()

	err = auth.Authenticate(next)(rec, req)
	if err == nil {
		t.Fatal("Authenticate() error = nil, want authn error")
	}

	line := strings.TrimSpace(logs.String())
	assertLogDoesNotContain(t, line, "secret-token")

	record := decodeLogRecord(t, line)
	if record[logFieldEvent] != "security.authn.denied" {
		t.Fatalf("event=%v want security.authn.denied", record[logFieldEvent])
	}
	if record[logFieldDecision] != logDecisionDenied {
		t.Fatalf("decision=%v want %s", record[logFieldDecision], logDecisionDenied)
	}
	if record[logFieldAuthNKind] != string(security.AuthNExpired) {
		t.Fatalf("authn_kind=%v want %s", record[logFieldAuthNKind], security.AuthNExpired)
	}
	if record[config.LogFieldRequestID] != "rid-authn" {
		t.Fatalf("request_id=%v want rid-authn", record[config.LogFieldRequestID])
	}
}

func TestAuthorize_LogsSafeDeniedReason(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil)).
		With(config.LogFieldComponent, logComponentHTTPSecurity)

	authz, err := NewAuthorizeAPIWithLogger([]security.RouteRule{
		{Pattern: "GET /admin", Access: security.AccessAdminOnly},
	}, logger)
	if err != nil {
		t.Fatalf("NewAuthorizeAPIWithLogger: %v", err)
	}

	next := httputils.AppHandler(func(http.ResponseWriter, *http.Request) error {
		t.Fatal("next handler was called")
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Pattern = "GET /admin"
	req = req.WithContext(requestid.WithRequestID(req.Context(), "rid-authz"))
	req = req.WithContext(security.WithPrincipal(req.Context(), security.Principal{
		UserID: 42,
		Role:   security.RoleUser,
	}))
	rec := httptest.NewRecorder()

	err = authz.Authorize(next)(rec, req)
	if err == nil {
		t.Fatal("Authorize() error = nil, want authz error")
	}

	line := strings.TrimSpace(logs.String())

	record := decodeLogRecord(t, line)
	if record[logFieldEvent] != "security.authz.denied" {
		t.Fatalf("event=%v want security.authz.denied", record[logFieldEvent])
	}
	if _, ok := record["user_id"]; ok {
		t.Fatalf("record=%#v must not include user_id", record)
	}
	if record[logFieldAuthZKind] != string(security.AuthZAdminRequired) {
		t.Fatalf("authz_kind=%v want %s", record[logFieldAuthZKind], security.AuthZAdminRequired)
	}
}

func TestCORS_LogsSafeDeniedReasonWithoutOriginOrHeaders(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil)).
		With(config.LogFieldComponent, logComponentHTTPSecurity)

	cors := NewCORSWithLogger(config.CORSConfig{
		AllowedOrigins: []string{"https://app.example.test"},
		AllowedMethods: []string{http.MethodGet},
		AllowedHeaders: []string{"Authorization", "X-Request-Id"},
	}, logger)

	h := cors.CORS(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler was called")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/users", nil)
	req.Header.Set("Origin", "https://evil.example.test")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "Authorization, X-Secret-Header")
	req = req.WithContext(requestid.WithRequestID(req.Context(), "rid-cors"))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}

	line := strings.TrimSpace(logs.String())
	assertLogDoesNotContain(t, line, "https://evil.example.test")
	assertLogDoesNotContain(t, line, "X-Secret-Header")
	assertLogDoesNotContain(t, line, "Authorization")

	record := decodeLogRecord(t, line)
	if record[logFieldEvent] != "security.cors.denied" {
		t.Fatalf("event=%v want security.cors.denied", record[logFieldEvent])
	}
	if record[logFieldCORSDenial] != "origin" {
		t.Fatalf("cors_denial=%v want origin", record[logFieldCORSDenial])
	}
}

func decodeLogRecord(t *testing.T, line string) map[string]any {
	t.Helper()

	var record map[string]any
	if err := json.Unmarshal([]byte(line), &record); err != nil {
		t.Fatalf("unmarshal log line: %v line=%q", err, line)
	}
	return record
}

func assertLogDoesNotContain(t *testing.T, logText, forbidden string) {
	t.Helper()

	if strings.Contains(logText, forbidden) {
		t.Fatalf("log=%q must not contain %q", logText, forbidden)
	}
}

type failingVerifier struct{}

func (failingVerifier) Verify(string) (security.Principal, error) {
	return security.Principal{}, &security.AuthNError{
		Kind:  security.AuthNExpired,
		Cause: errors.New("expired"),
	}
}

var _ security.Verifier = failingVerifier{}
