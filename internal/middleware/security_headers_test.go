package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"pet-study/internal/config"
	"pet-study/internal/security"
)

func TestSecurityHeaders_On200_HTTP_NoHSTS(t *testing.T) {
	sec := NewSecurityHeaders(config.SecurityHeadersConfig{
		Enable:         true,
		ReferrerPolicy: "no-referrer",
		HSTS: config.HSTSConfig{
			Enable: true,
			MaxAge: 31536000,
		},
	})

	h := sec.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/1", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options=%q want=%q", got, "nosniff")
	}
	if got := rec.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy=%q want=%q", got, "no-referrer")
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options=%q want=%q", got, "DENY")
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("HSTS must be empty on plain http, got=%q", got)
	}
}

func TestSecurityHeaders_HSTS_OnEffectiveHTTPS(t *testing.T) {
	sec := NewSecurityHeaders(config.SecurityHeadersConfig{
		Enable:         true,
		ReferrerPolicy: "no-referrer",
		HSTS: config.HSTSConfig{
			Enable: true,
			MaxAge: 31536000,
		},
	})

	h := sec.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/1", nil)
	req = req.WithContext(security.WithRequestInfo(req.Context(), security.RequestInfo{
		ClientIP: "203.0.113.10",
		Scheme:   "https",
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got == "" {
		t.Fatalf("missing HSTS on effective https")
	}
}

func TestSecurityHeaders_HSTS_OnDirectTLS(t *testing.T) {
	sec := NewSecurityHeaders(config.SecurityHeadersConfig{
		Enable:         true,
		ReferrerPolicy: "no-referrer",
		HSTS: config.HSTSConfig{
			Enable: true,
			MaxAge: 31536000,
		},
	})

	h := sec.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/1", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got == "" {
		t.Fatalf("missing HSTS on direct TLS")
	}
}
