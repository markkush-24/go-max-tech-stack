package middleware

import (
	"encoding/json"
	"mime"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"pet-study/internal/httputils"
	"pet-study/internal/requestid"
	"pet-study/internal/security"
)

type expiredVerifier struct{}

func (expiredVerifier) Verify(_ string) (security.Principal, error) {
	return security.Principal{}, jwt.ErrTokenExpired
}

type authProblem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail"`
	Instance  string `json:"instance"`
	RequestID string `json:"request_id"`
}

func TestAuth_ExpiredToken_401(t *testing.T) {
	auth := NewAuthAPI(expiredVerifier{})

	app := httputils.AppHandler(func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusNoContent)
		return nil
	})

	h := requestid.RequestIDMiddleware(auth.Authenticate(app))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/1", nil)
	req.Header.Set("Authorization", "Bearer test-expired-token")
	req.Header.Set("Accept", "application/json")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}

	mt, _, err := mime.ParseMediaType(rec.Header().Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse media type: %v", err)
	}
	if mt != "application/problem+json" {
		t.Fatalf("Content-Type=%q want=application/problem+json", mt)
	}

	if got := rec.Header().Get("WWW-Authenticate"); got == "" {
		t.Fatalf("missing WWW-Authenticate")
	}
	if got := rec.Header().Get(requestid.HeaderName); got == "" {
		t.Fatalf("missing %s", requestid.HeaderName)
	}

	var p authProblem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode problem: %v body=%s", err, rec.Body.String())
	}
	if p.Status != http.StatusUnauthorized {
		t.Fatalf("problem.status=%d want=%d", p.Status, http.StatusUnauthorized)
	}
	if p.RequestID == "" {
		t.Fatalf("problem.request_id is empty")
	}
}
