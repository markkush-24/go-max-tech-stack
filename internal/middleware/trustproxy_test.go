package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"pet-study/internal/config"
	"pet-study/internal/requestid"
	"pet-study/internal/security"
)

func TestTrustProxy_Untrusted_IgnoresXFFAndXFP(t *testing.T) {
	proxyAPI, err := NewProxyAPI(config.ProxyConfig{
		TrustedProxies: []netip.Prefix{
			netip.MustParsePrefix("127.0.0.1/32"),
		},
		TrustXFF: true,
		TrustXFP: true,
	})
	if err != nil {
		t.Fatalf("NewProxyAPI: %v", err)
	}

	h := proxyAPI.TrustProxy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ri, ok := security.RequestInfoFromContext(r.Context())
		if !ok {
			t.Fatalf("missing RequestInfo in context")
		}
		w.Header().Set("X-Test-Client-IP", ri.ClientIP)
		w.Header().Set("X-Test-Scheme", ri.Scheme)
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/1", nil)
	req.RemoteAddr = "203.0.113.10:34567"
	req.Header.Set("X-Forwarded-For", "198.51.100.7")
	req.Header.Set("X-Forwarded-Proto", "https")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d want=%d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("X-Test-Client-IP"); got != "203.0.113.10" {
		t.Fatalf("clientIP=%q want=%q", got, "203.0.113.10")
	}
	if got := rec.Header().Get("X-Test-Scheme"); got != "http" {
		t.Fatalf("scheme=%q want=%q", got, "http")
	}
}

func TestTrustProxy_Trusted_UsesXFFAndXFP(t *testing.T) {
	proxyAPI, err := NewProxyAPI(config.ProxyConfig{
		TrustedProxies: []netip.Prefix{
			netip.MustParsePrefix("127.0.0.1/32"),
		},
		TrustXFF: true,
		TrustXFP: true,
	})
	if err != nil {
		t.Fatalf("NewProxyAPI: %v", err)
	}

	h := proxyAPI.TrustProxy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ri, ok := security.RequestInfoFromContext(r.Context())
		if !ok {
			t.Fatalf("missing RequestInfo in context")
		}
		w.Header().Set("X-Test-Client-IP", ri.ClientIP)
		w.Header().Set("X-Test-Scheme", ri.Scheme)
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/1", nil)
	req.RemoteAddr = "127.0.0.1:34567"
	req.Header.Set("X-Forwarded-For", "198.51.100.7")
	req.Header.Set("X-Forwarded-Proto", "https")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d want=%d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("X-Test-Client-IP"); got != "198.51.100.7" {
		t.Fatalf("clientIP=%q want=%q", got, "198.51.100.7")
	}
	if got := rec.Header().Get("X-Test-Scheme"); got != "https" {
		t.Fatalf("scheme=%q want=%q", got, "https")
	}
}

func TestSanitizeRequestIDHeader_Untrusted_RemovesHeader(t *testing.T) {
	proxyAPI, err := NewProxyAPI(config.ProxyConfig{
		TrustedProxies: []netip.Prefix{
			netip.MustParsePrefix("127.0.0.1/32"),
		},
		TrustXFF: true,
		TrustXFP: true,
	})
	if err != nil {
		t.Fatalf("NewProxyAPI: %v", err)
	}

	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid, _ := requestid.RequestID(r.Context())
		w.Header().Set("X-Echo-Request-Id", rid)
		w.WriteHeader(http.StatusNoContent)
	})

	h := proxyAPI.SanitizeRequestIDHeader(requestid.RequestIDMiddleware(final))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/1", nil)
	req.RemoteAddr = "203.0.113.10:34567"
	req.Header.Set(requestid.HeaderName, "attacker-123")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d want=%d", rec.Code, http.StatusNoContent)
	}
	got := rec.Header().Get("X-Echo-Request-Id")
	if got == "" {
		t.Fatalf("missing echoed request id")
	}
	if got == "attacker-123" {
		t.Fatalf("request id was not sanitized")
	}
}

func TestSanitizeRequestIDHeader_Trusted_PreservesHeader(t *testing.T) {
	proxyAPI, err := NewProxyAPI(config.ProxyConfig{
		TrustedProxies: []netip.Prefix{
			netip.MustParsePrefix("127.0.0.1/32"),
		},
		TrustXFF: true,
		TrustXFP: true,
	})
	if err != nil {
		t.Fatalf("NewProxyAPI: %v", err)
	}

	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid, _ := requestid.RequestID(r.Context())
		w.Header().Set("X-Echo-Request-Id", rid)
		w.WriteHeader(http.StatusNoContent)
	})

	h := proxyAPI.SanitizeRequestIDHeader(requestid.RequestIDMiddleware(final))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/1", nil)
	req.RemoteAddr = "127.0.0.1:34567"
	req.Header.Set(requestid.HeaderName, "proxy-rid-123")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d want=%d", rec.Code, http.StatusNoContent)
	}
	got := rec.Header().Get("X-Echo-Request-Id")
	if got != "proxy-rid-123" {
		t.Fatalf("request id=%q want=%q", got, "proxy-rid-123")
	}
}
