package router_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"pet-study/internal/entity"
	"pet-study/internal/metrics"
	"pet-study/internal/middleware"
	"pet-study/internal/outbound/profile"
	"pet-study/internal/queue"
	"pet-study/internal/requestid"
	apirouter "pet-study/internal/router"
	routes "pet-study/internal/routes"
	"pet-study/internal/security"
	"pet-study/internal/service"
	"pet-study/internal/store/jobrepo"
	"pet-study/internal/store/userrepo"
	"pet-study/internal/stream"
	"pet-study/internal/testkit"
	"testing"
	"time"
)

type tokenMapVerifier struct {
	tokens map[string]security.Principal
}

func (v tokenMapVerifier) Verify(token string) (security.Principal, error) {
	if p, ok := v.tokens[token]; ok {
		return p, nil
	}
	return security.Principal{}, &security.AuthNError{Kind: security.AuthNInvalid}
}

type stubProfileClient struct {
	seenRID string
}

func (c *stubProfileClient) FetchProfile(_ context.Context, userID int64, requestID string) (profile.Profile, error) {
	c.seenRID = requestID
	return profile.Profile{UserID: userID, Bio: "bio", City: "nyc"}, nil
}

type problemWithRID struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail"`
	Instance  string `json:"instance"`
	RequestID string `json:"request_id"`
}

func decodeProblemRID(t *testing.T, rec *httptest.ResponseRecorder) problemWithRID {
	t.Helper()
	var p problemWithRID
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("unmarshal problem: %v, body=%q", err, rec.Body.String())
	}
	return p
}

type env struct {
	h http.Handler
}

func newEnv(t *testing.T) (*env, *service.UserService) {
	t.Helper()

	userRepo := userrepo.NewMemoryUserRepository()
	jobRepo := jobrepo.NewMemoryJobRepository()

	userSvc := service.NewUserService(userRepo)
	jobSvc := service.NewJobService(jobRepo)
	eventHub := stream.NewHub(16)

	m := metrics.DefaultHTTP()
	q := queue.New(10)
	hub := stream.NewHub(16)
	v1 := routes.NewUserHandler(userSvc, jobSvc, q, m, hub)
	v2 := routes.NewUserV2Handler(userSvc, jobSvc, q, m, hub)
	jh := routes.NewJobHandler(jobSvc, eventHub, 5*time.Second, 5*time.Second)

	// profile endpoint: подключаем реальный handler с stub client
	pc := &stubProfileClient{}
	profileSvc := service.NewUserProfileService(userSvc, pc, 200*time.Millisecond)
	profileH := routes.NewUsersProfileHandler(profileSvc)

	lim := middleware.NewRateLimitedAPI(1000, 1000)
	bh := middleware.NewBulkhead(1000)

	ver := tokenMapVerifier{tokens: map[string]security.Principal{
		"user1": {UserID: 1, Role: security.RoleUser},
		"admin": {UserID: 999, Role: security.RoleAdmin},
	}}

	auth, err := middleware.NewAuthAPI(ver)
	if err != nil {
		t.Fatalf("NewAuthAPI: %v", err)
	}
	rbac, err := middleware.NewAuthorizeAPI(security.DefaultPolicy)
	if err != nil {
		t.Fatalf("NewAuthorizeAPI: %v", err)
	}

	api := apirouter.NewRouter(v1, v2, jh, profileH, lim, bh, auth, rbac)

	// важное: request-id должен быть наружу, чтобы Problem включал request_id
	h := requestid.RequestIDMiddleware(api)

	return &env{h: h}, userSvc
}

func authReq(method, path, token string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	return req
}

func TestUsersGetByID_SelfAllowed_200(t *testing.T) {
	e, userSvc := newEnv(t)

	// создаём user id=1
	_, err := userSvc.CreateUser(t.Context(), &entity.CreateUserInput{Name: "Bob", Email: "bob@example.com", Age: 21})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, authReq("GET", "/api/v1/users/1", "user1"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want=200 body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get(requestid.HeaderName) == "" {
		t.Fatalf("missing %s header", requestid.HeaderName)
	}
	if rec.Header().Get("ETag") == "" {
		t.Fatalf("missing ETag")
	}
}

func TestUsersGetByID_ForeignForbidden_403(t *testing.T) {
	e, userSvc := newEnv(t)

	// создаём user id=1 и id=2
	_, err := userSvc.CreateUser(t.Context(), &entity.CreateUserInput{Name: "Bob", Email: "bob@example.com", Age: 21})
	if err != nil {
		t.Fatalf("create user1: %v", err)
	}
	_, err = userSvc.CreateUser(t.Context(), &entity.CreateUserInput{Name: "Alice", Email: "alice@example.com", Age: 22})
	if err != nil {
		t.Fatalf("create user2: %v", err)
	}

	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, authReq("GET", "/api/v1/users/2", "user1"))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=403 body=%s", rec.Code, rec.Body.String())
	}
	if got := testkit.MustMediaType(t, rec); got != "application/problem+json" {
		t.Fatalf("mediaType=%q want=application/problem+json", got)
	}
	p := decodeProblemRID(t, rec)
	if p.Status != 403 || p.RequestID == "" {
		t.Fatalf("problem.status=%d request_id=%q body=%s", p.Status, p.RequestID, rec.Body.String())
	}
}

func TestUsersGetByID_AdminAllowed_200(t *testing.T) {
	e, userSvc := newEnv(t)

	// создаём user id=2
	_, _ = userSvc.CreateUser(t.Context(), &entity.CreateUserInput{Name: "Bob", Email: "bob@example.com", Age: 21})
	_, err := userSvc.CreateUser(t.Context(), &entity.CreateUserInput{Name: "Alice", Email: "alice@example.com", Age: 22})
	if err != nil {
		t.Fatalf("create user2: %v", err)
	}

	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, authReq("GET", "/api/v1/users/2", "admin"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want=200 body=%s", rec.Code, rec.Body.String())
	}
}

func TestUsersGetByID_ETag_NotModified_304(t *testing.T) {
	e, userSvc := newEnv(t)

	_, err := userSvc.CreateUser(t.Context(), &entity.CreateUserInput{Name: "Bob", Email: "bob@example.com", Age: 21})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// 1) получаем ETag
	rec1 := httptest.NewRecorder()
	e.h.ServeHTTP(rec1, authReq("GET", "/api/v1/users/1", "user1"))
	if rec1.Code != http.StatusOK {
		t.Fatalf("status=%d want=200 body=%s", rec1.Code, rec1.Body.String())
	}
	etag := rec1.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("missing ETag")
	}

	// 2) If-None-Match -> 304
	req2 := authReq("GET", "/api/v1/users/1", "user1")
	req2.Header.Set("If-None-Match", etag)

	rec2 := httptest.NewRecorder()
	e.h.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusNotModified {
		t.Fatalf("status=%d want=304 body=%s", rec2.Code, rec2.Body.String())
	}
	if got := rec2.Header().Get("ETag"); got != etag {
		t.Fatalf("ETag=%q want=%q", got, etag)
	}
}

func TestUsersProfile_ForeignForbidden_403(t *testing.T) {
	e, userSvc := newEnv(t)

	_, _ = userSvc.CreateUser(t.Context(), &entity.CreateUserInput{Name: "Bob", Email: "bob@example.com", Age: 21})
	_, _ = userSvc.CreateUser(t.Context(), &entity.CreateUserInput{Name: "Alice", Email: "alice@example.com", Age: 22})

	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, authReq("GET", "/api/v1/users/2/profile", "user1"))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=403 body=%s", rec.Code, rec.Body.String())
	}
}
