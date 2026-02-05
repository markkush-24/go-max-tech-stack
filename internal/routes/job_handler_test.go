package routes

import (
	"bytes"
	"context"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"pet-study/internal/health"
	"pet-study/internal/metrics"
	"pet-study/internal/middleware"
	"pet-study/internal/queue"
	"pet-study/internal/requestid"
	"pet-study/internal/router"
	"pet-study/internal/service"
	"pet-study/internal/store/jobrepo"
	"pet-study/internal/store/userrepo"
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

	q := queue.New(1)
	pool := workerpool.NewWorkerPool(q, jobSvc, userSvc)
	err := pool.Start(0)
	if err != nil {
		t.Fatalf("start workerpool: %v", err)
	}

	readiness := health.NewReadiness(
		health.Check{Name: "repo", Fn: userRepo.Ping},
		health.Check{Name: "workerpool", Fn: pool.CheckRunning})

	v1 := NewUserHandler(userSvc, jobSvc, q)
	v2 := NewUserV2Handler(userSvc, jobSvc, q)

	lim := middleware.NewRateLimitedAPI(float64(10), 5)

	jh := NewJobHandler(jobSvc)

	userRouter := router.NewRouter(v1, v2, jh, lim)

	healthRouter := router.NewHealthRouter(readiness)
	rootRouter := router.NewRoot(userRouter, healthRouter, nil)

	// Metrics registry
	m := metrics.DefaultHTTP()

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

	resp, err := server.Client().Post(
		server.URL+"/api/v1/users?async=1",
		"application/json",
		bytes.NewBufferString(`{"name":"bob","email":"bob@example.com","age":21}`),
	)
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

	resp2, errUnavailable := server.Client().Post(
		server.URL+"/api/v1/users?async=1",
		"application/json",
		bytes.NewBufferString(`{"name":"bob","email":"bob@example.com","age":21}`),
	)

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
