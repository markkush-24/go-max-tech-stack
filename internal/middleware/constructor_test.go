package middleware

import (
	"strings"
	"testing"

	"pet-study/internal/security"
)

func TestNewAuthAPI_ReturnsErrorOnNilVerifier(t *testing.T) {
	auth, err := NewAuthAPI(nil)
	if err == nil {
		t.Fatal("expected error for nil verifier")
	}
	if auth != nil {
		t.Fatalf("auth=%v want=nil", auth)
	}
	if !strings.Contains(err.Error(), "verifier is nil") {
		t.Fatalf("err=%q must mention verifier", err)
	}
}

func TestNewAuthorizeAPI_ReturnsErrorOnInvalidPolicy(t *testing.T) {
	t.Run("empty policy", func(t *testing.T) {
		authz, err := NewAuthorizeAPI(nil)
		if err == nil {
			t.Fatal("expected error for empty policy")
		}
		if authz != nil {
			t.Fatalf("authz=%v want=nil", authz)
		}
	})

	t.Run("empty pattern", func(t *testing.T) {
		authz, err := NewAuthorizeAPI([]security.RouteRule{{Access: security.AccessPublic}})
		if err == nil {
			t.Fatal("expected error for empty pattern")
		}
		if authz != nil {
			t.Fatalf("authz=%v want=nil", authz)
		}
	})

	t.Run("duplicate pattern", func(t *testing.T) {
		policy := []security.RouteRule{
			{Pattern: "GET /x", Access: security.AccessPublic},
			{Pattern: "GET /x", Access: security.AccessAuthenticated},
		}

		authz, err := NewAuthorizeAPI(policy)
		if err == nil {
			t.Fatal("expected error for duplicate pattern")
		}
		if authz != nil {
			t.Fatalf("authz=%v want=nil", authz)
		}
		if !strings.Contains(err.Error(), `duplicate rule pattern "GET /x"`) {
			t.Fatalf("err=%q must mention duplicate pattern", err)
		}
	})
}
