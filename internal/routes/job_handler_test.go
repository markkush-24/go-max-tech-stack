package routes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"pet-study/internal/health"
	"pet-study/internal/metrics"
	"pet-study/internal/middleware"
	"pet-study/internal/queue"
	"pet-study/internal/requestid"
	apirouter "pet-study/internal/router"
	"pet-study/internal/routes"
	"pet-study/internal/security"
	"pet-study/internal/service"
	"pet-study/internal/store/jobrepo"
	"pet-study/internal/store/userrepo"
	"pet-study/internal/stream"
	"pet-study/internal/testkit"
	"pet-study/internal/workerpool"
	"time"

	"testing"
)

func TestQueueOverflowFastFail(t *testing.T) {
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	userRepo := userrepo.NewMemoryUserRepository()
	jobRepo := jobrepo.NewMemoryJobRepository()

	userSvc := service.NewUserService(userRepo)
	jobSvc := service.NewJobService(jobRepo)

	// Metrics registry
	m := metrics.DefaultHTTP()

	hub := stream.NewHub(16)
	q := queue.New(1)
	pool := workerpool.NewWorkerPool(q, jobSvc, userSvc, m, hub)
	err := pool.Start(context.Background(), 0)
	if err != nil {
		t.Fatalf("start workerpool: %v", err)
	}

	readiness := health.NewReadiness(
		health.Check{Name: "repo", Fn: userRepo.Ping},
		health.Check{Name: "workerpool", Fn: pool.CheckRunning})

	v1 := routes.NewUserHandler(userSvc, jobSvc, q, m, hub)
	v2 := routes.NewUserV2Handler(userSvc, jobSvc, q, m, hub)

	lim := middleware.NewRateLimitedAPI(float64(10), 5)
	bh := middleware.NewBulkhead(1)
	auth, err := middleware.NewAuthAPI(testkit.StubVerifier{P: security.Principal{UserID: 1, Role: security.RoleAdmin}})
	if err != nil {
		t.Fatalf("NewAuthAPI: %v", err)
	}
	rbac, err := middleware.NewAuthorizeAPI(security.DefaultPolicy)
	if err != nil {
		t.Fatalf("NewAuthorizeAPI: %v", err)
	}

	eventHub := stream.NewHub(16)
	jh := routes.NewJobHandler(jobSvc, eventHub, 5*time.Second, 5*time.Second, nil)

	userRouter := apirouter.NewRouter(v1, v2, jh, nil, lim, bh, auth, rbac)

	healthRouter := apirouter.NewHealthRouter(readiness)
	rootRouter := apirouter.NewRoot(userRouter, healthRouter, nil)

	// Middleware chain (outer -> inner):
	// RequestID -> Metrics -> Logger -> Recover -> Router
	handler := rootRouter
	handler = middleware.Recover(handler) // inner: чтобы Logger/Metrics увидели 500 при panic в Router
	handler = middleware.Logger(handler)
	handler = middleware.Metrics(m)(handler)
	handler = middleware.Recover(handler) // outer: ловит panic в Logger/Metrics
	handler = requestid.RequestIDMiddleware(handler)

	server := httptest.NewServer(handler)

	t.Cleanup(server.Close)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := pool.Stop(ctx); err != nil {
			t.Errorf("stop workerpool: %v", err)
		}
	})

	client := server.Client()

	req, _ := http.NewRequest("POST", server.URL+"/api/v1/users?async=1",
		bytes.NewBufferString(`{"name":"bob","email":"bob@example.com","age":21}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test")

	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("post #1: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("post #1 status=%d want=%d body=%s", resp.StatusCode, http.StatusAccepted, b)
	}

	if loc := resp.Header.Get("Location"); loc == "" {
		t.Fatalf("missing Location on 202")
	}

	if rid := resp.Header.Get(requestid.HeaderName); rid == "" {
		t.Fatalf("missing %s header", requestid.HeaderName)
	}

	req2, _ := http.NewRequest("POST", server.URL+"/api/v1/users?async=1",
		bytes.NewBufferString(`{"name":"bob","email":"bob@example.com","age":21}`),
	)
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer test")

	resp2, errUnavailable := client.Do(req2)

	if errUnavailable != nil {
		t.Fatalf("post #2: %v", errUnavailable)
	}

	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusServiceUnavailable {
		b, _ := io.ReadAll(resp2.Body)
		t.Fatalf("post #2 status=%d want=%d body=%s", resp2.StatusCode, http.StatusServiceUnavailable, b)
	}

	if got := resp2.Header.Get("Location"); got != "" {
		t.Fatalf("Location=%q want=%q", got, "")
	}

	rawCT := resp2.Header.Get("Content-Type")
	mt, _, err := mime.ParseMediaType(rawCT)
	if err != nil {
		t.Fatalf("bad Content-Type %q: %v", rawCT, err)
	}
	if mt != "application/problem+json" {
		t.Fatalf("Content-Type=%q want=%q", mt, "application/problem+json")
	}

	rid := resp2.Header.Get(requestid.HeaderName)
	if rid == "" {
		t.Fatalf("%s empty", requestid.HeaderName)
	}

	all, errJobRepo := jobRepo.GetAll(ctxWithTimeout)
	if errJobRepo != nil {
		t.Fatalf("jobRepo.GetAll: %v", errJobRepo)
	}
	jobCount := len(all)
	if jobCount != 1 {
		t.Fatalf("jobRepo must contain only one job, current count: %d", jobCount)
	}
}

func TestJobNotFound(t *testing.T) {
	userRepo := userrepo.NewMemoryUserRepository()
	userSvc := service.NewUserService(userRepo)
	jobRepo := jobrepo.NewMemoryJobRepository()
	jobSvc := service.NewJobService(jobRepo)

	// Metrics registry
	m := metrics.DefaultHTTP()

	q := queue.New(1)

	hub := stream.NewHub(16)
	v1 := routes.NewUserHandler(userSvc, jobSvc, q, m, hub)
	v2 := routes.NewUserV2Handler(userSvc, jobSvc, q, m, hub)

	lim := middleware.NewRateLimitedAPI(float64(10), 5)
	bh := middleware.NewBulkhead(1)

	auth, err := middleware.NewAuthAPI(testkit.StubVerifier{P: security.Principal{UserID: 1, Role: security.RoleAdmin}})
	if err != nil {
		t.Fatalf("NewAuthAPI: %v", err)
	}
	rbac, err := middleware.NewAuthorizeAPI(security.DefaultPolicy)
	if err != nil {
		t.Fatalf("NewAuthorizeAPI: %v", err)
	}

	eventHub := stream.NewHub(16)
	jh := routes.NewJobHandler(jobSvc, eventHub, 5*time.Second, 5*time.Second, nil)

	userRouter := apirouter.NewRouter(v1, v2, jh, nil, lim, bh, auth, rbac)

	//healthRouter := router.NewHealthRouter(readiness)
	rootRouter := apirouter.NewRoot(userRouter, http.NewServeMux(), nil)

	// Middleware chain (outer -> inner):
	// RequestID -> Metrics -> Logger -> Recover -> Router
	handler := rootRouter
	handler = middleware.Recover(handler) // inner: чтобы Logger/Metrics увидели 500 при panic в Router
	handler = middleware.Logger(handler)
	handler = middleware.Metrics(m)(handler)
	handler = middleware.Recover(handler) // outer: ловит panic в Logger/Metrics
	handler = requestid.RequestIDMiddleware(handler)

	server := httptest.NewServer(handler)

	t.Cleanup(server.Close)

	client := server.Client()
	req, _ := http.NewRequest("GET", server.URL+"/api/v1/jobs/999999", nil)
	req.Header.Set("Authorization", "Bearer test")
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("get #1: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("get status=%d want=%d", resp.StatusCode, http.StatusNotFound)
	}

	rawCT := resp.Header.Get("Content-Type")
	mt, _, err := mime.ParseMediaType(rawCT)
	if err != nil {
		t.Fatalf("bad Content-Type %q: %v", rawCT, err)
	}
	if mt != "application/problem+json" {
		t.Fatalf("Content-Type=%q want=%q", mt, "application/problem+json")
	}

	rid := resp.Header.Get(requestid.HeaderName)
	if rid == "" {
		t.Fatalf("%s empty", requestid.HeaderName)
	}

	var p struct {
		Status int `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if p.Status != http.StatusNotFound {
		t.Fatalf("problem.status=%d want=%d", p.Status, http.StatusNotFound)
	}
}
