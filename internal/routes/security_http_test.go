package router_test

import (
	"encoding/json"
	"mime"
	"net/http"
	"net/http/httptest"
	"testing"

	"pet-study/internal/requestid"
	"pet-study/internal/testkit"
)

type problem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail"`
	Instance  string `json:"instance"`
	RequestID string `json:"request_id"`
}

func decodeProblem(t *testing.T, rec *httptest.ResponseRecorder) problem {
	t.Helper()
	var p problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode problem: %v body=%s", err, rec.Body.String())
	}
	return p
}

func mustMediaType(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	mt, _, err := mime.ParseMediaType(rec.Header().Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse media type: %v", err)
	}
	return mt
}

func TestSecurity_401_MissingToken(t *testing.T) {
	h, _ := testkit.NewUserRouter(t, testkit.WithoutAuthInjection())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/1", nil)
	req.Header.Set("Accept", "application/json")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	if got := mustMediaType(t, rec); got != "application/problem+json" {
		t.Fatalf("Content-Type=%q want=application/problem+json", got)
	}
	if got := rec.Header().Get("WWW-Authenticate"); got == "" {
		t.Fatalf("missing WWW-Authenticate")
	}
	if got := rec.Header().Get(requestid.HeaderName); got == "" {
		t.Fatalf("missing %s header", requestid.HeaderName)
	}

	p := decodeProblem(t, rec)
	if p.Status != http.StatusUnauthorized {
		t.Fatalf("problem.status=%d want=%d", p.Status, http.StatusUnauthorized)
	}
	if p.RequestID == "" {
		t.Fatalf("problem.request_id is empty")
	}
}

func TestSecurity_401_InvalidToken(t *testing.T) {
	h, _ := testkit.NewUserRouter(t, testkit.WithoutAuthInjection())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/1", nil)
	req.Header.Set("Authorization", "Bearer definitely-not-a-jwt")
	req.Header.Set("Accept", "application/json")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Fatalf("missing WWW-Authenticate")
	}
}

func TestSecurity_403_RBAC_UserCannotReadJobs(t *testing.T) {
	h, _ := testkit.NewUserRouter(
		t,
		testkit.WithoutAuthInjection(),
		testkit.WithPrincipalUser(1), // helper надо добавить в testkit
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/999999", nil)
	req.Header.Set("Authorization", "Bearer test")
	req.Header.Set("Accept", "application/json")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	p := decodeProblem(t, rec)
	if p.Status != http.StatusForbidden || p.RequestID == "" {
		t.Fatalf("problem=%+v", p)
	}
}

func TestSecurity_403_ResourceLevel_UserCannotReadForeignUser(t *testing.T) {
	h, app := testkit.NewUserRouter(
		t,
		testkit.WithoutAuthInjection(),
		testkit.WithPrincipalUser(1),
	)

	_, _ = app.UserSvc.CreateUser(testkit.ReqContext(), testkit.CreateUser("Bob", "bob@example.com", 21))     // id=1
	_, _ = app.UserSvc.CreateUser(testkit.ReqContext(), testkit.CreateUser("Alice", "alice@example.com", 22)) // id=2

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/2", nil)
	req.Header.Set("Authorization", "Bearer test")
	req.Header.Set("Accept", "application/json")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestSecurity_CORS_Preflight_204(t *testing.T) {
	h, _ := testkit.NewUserRouter(
		t,
		testkit.WithCORSAllowlist("http://localhost:3000"), // helper надо добавить
	)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/users", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "authorization, content-type")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("ACAO=%q want=%q", got, "http://localhost:3000")
	}
	if got := rec.Header().Get("Vary"); got == "" {
		t.Fatalf("missing Vary")
	}
}

func TestSecurity_CORS_DenyOrigin_403(t *testing.T) {
	h, _ := testkit.NewUserRouter(
		t,
		testkit.WithCORSAllowlist("http://localhost:3000"),
		testkit.WithPrincipalAdmin(1),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/1", nil)
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Authorization", "Bearer test")
	req.Header.Set("Accept", "application/json")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestSecurity_Headers_On404(t *testing.T) {
	h, _ := testkit.NewUserRouter(
		t,
		testkit.WithPrincipalAdmin(1),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/999999", nil)
	req.Header.Set("Authorization", "Bearer test")
	req.Header.Set("Accept", "application/json")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options=%q want=nosniff", got)
	}
	if got := rec.Header().Get("Referrer-Policy"); got == "" {
		t.Fatalf("missing Referrer-Policy")
	}
	if got := rec.Header().Get("X-Frame-Options"); got == "" {
		t.Fatalf("missing X-Frame-Options")
	}
}
