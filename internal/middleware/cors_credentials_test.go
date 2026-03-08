package middleware

import (
	"mime"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pet-study/internal/config"
)

func TestCORS_CredentialsWithWildcardOrigin_Denied(t *testing.T) {
	cors := NewCORS(config.CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "If-None-Match", "X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           5 * time.Minute,
	})

	nextCalled := false
	h := cors.CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/1", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if nextCalled {
		t.Fatalf("next handler must not be called")
	}

	mt, _, err := mime.ParseMediaType(rec.Header().Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse media type: %v", err)
	}
	if mt != "application/problem+json" {
		t.Fatalf("Content-Type=%q want=application/problem+json", mt)
	}
}
