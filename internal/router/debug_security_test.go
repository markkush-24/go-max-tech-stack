package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"pet-study/internal/httputils"
	"pet-study/internal/middleware"
	"pet-study/internal/requestid"
	"pet-study/internal/router"
	"pet-study/internal/security"
	"pet-study/internal/testkit"
)

func newDebugServer(t *testing.T, principal security.Principal, injectAuth bool) *httptest.Server {
	t.Helper()

	rawDebug := router.NewDebugRouter()

	ver := testkit.StubVerifier{P: principal}
	auth := middleware.NewAuthAPI(ver)
	rbac := middleware.NewAuthorizeAPI(security.DefaultPolicy)

	dbg := httputils.HandlerToApp(rawDebug)
	dbg = rbac.Authorize(dbg)
	dbg = auth.Authenticate(dbg)

	root := router.NewRoot(http.NewServeMux(), http.NewServeMux(), dbg)

	var h http.Handler = root
	h = middleware.Recover(h)
	h = middleware.Logger(h)
	h = middleware.Metrics(testkit.NewMetricsForTests())(h)
	h = middleware.Recover(h)
	h = requestid.RequestIDMiddleware(h)

	if injectAuth {
		h = testkit.InjectBearerForTests("test", h)
	}

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

func TestDebug_NoToken_Unauthorized(t *testing.T) {
	srv := newDebugServer(t, security.Principal{UserID: 1, Role: security.RoleAdmin}, false)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/debug/vars", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", resp.StatusCode, http.StatusUnauthorized)
	}
	if resp.Header.Get("WWW-Authenticate") == "" {
		t.Fatalf("missing WWW-Authenticate")
	}
	if resp.Header.Get(requestid.HeaderName) == "" {
		t.Fatalf("missing %s", requestid.HeaderName)
	}
}

func TestDebug_User_Forbidden(t *testing.T) {
	srv := newDebugServer(t, security.Principal{UserID: 1, Role: security.RoleUser}, true)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/debug/vars", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status=%d want=%d", resp.StatusCode, http.StatusForbidden)
	}
	if resp.Header.Get(requestid.HeaderName) == "" {
		t.Fatalf("missing %s", requestid.HeaderName)
	}
}

func TestDebug_Admin_OK(t *testing.T) {
	srv := newDebugServer(t, security.Principal{UserID: 1, Role: security.RoleAdmin}, true)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/debug/vars", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d want=%d", resp.StatusCode, http.StatusOK)
	}
	if resp.Header.Get(requestid.HeaderName) == "" {
		t.Fatalf("missing %s", requestid.HeaderName)
	}
}
