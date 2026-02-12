package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"expvar"
	"fmt"
	"net/http"
	"net/http/httptest"
	"pet-study/internal/entity"
	"pet-study/internal/health"
	"pet-study/internal/metrics"
	"pet-study/internal/middleware"
	"pet-study/internal/queue"
	"pet-study/internal/requestid"
	"pet-study/internal/router"
	"pet-study/internal/routes"
	"pet-study/internal/service"
	"pet-study/internal/store/jobrepo"
	"pet-study/internal/store/userrepo"
	"pet-study/internal/workerpool"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestIncreaseJobAsyncMetricsWithLatency(t *testing.T) {
	userRepo := userrepo.NewMemoryUserRepository()
	jobRepo := jobrepo.NewMemoryJobRepository()

	userSvc := service.NewUserService(userRepo)
	jobSvc := service.NewJobService(jobRepo)

	// Metrics registry
	m := metrics.DefaultHTTP()

	q := queue.New(1)
	pool := workerpool.NewWorkerPool(q, jobSvc, userSvc, m)
	err := pool.Start(1)
	if err != nil {
		t.Fatalf("start workerpool: %v", err)
	}

	readiness := health.NewReadiness(
		health.Check{Name: "repo", Fn: userRepo.Ping},
		health.Check{Name: "workerpool", Fn: pool.CheckRunning})

	v1 := routes.NewUserHandler(userSvc, jobSvc, q, m)
	v2 := routes.NewUserV2Handler(userSvc, jobSvc, q, m)

	lim := middleware.NewRateLimitedAPI(1e9, 1e9)
	bh := middleware.NewBulkhead(100)

	jh := routes.NewJobHandler(jobSvc)

	userRouter := router.NewRouter(v1, v2, jh, lim, bh)

	healthRouter := router.NewHealthRouter(readiness)
	rootRouter := router.NewRoot(userRouter, healthRouter, nil)

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

	queueBefore := getJobsTotal(t, "queued")
	expectedQueue := queueBefore + 1

	runningBefore := getJobsTotal(t, "running")
	expectedRunning := runningBefore + 1

	succeededBefore := getJobsTotal(t, "succeeded")
	expectedSucceeded := succeededBefore + 1

	latencyBefore := expvar.Get("job_processing_latency_ns_count").(*expvar.Int).Value()
	expectedLatency := latencyBefore + 1

	body := fmt.Sprintf(`{"name":"bob","email":"bob%d@example.com","age":21}`, time.Now().UnixNano())
	resp, err := server.Client().Post(
		server.URL+"/api/v1/users?async=1",
		"application/json",
		bytes.NewBufferString(body),
	)
	if err != nil {
		t.Fatalf("get #1: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("get #1 status=%d want=%d", resp.StatusCode, http.StatusAccepted)
	}

	loc := resp.Header.Get("Location")
	if loc == "" {
		t.Fatal("missing Location header")
	}

	idStr := strings.TrimPrefix(loc, "/api/v1/jobs/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		t.Fatalf("parse job id from Location=%q: %v", loc, err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		r, err := server.Client().Get(server.URL + loc)
		if err != nil {
			t.Fatalf("get job: %v", err)
		}
		var job entity.Job
		err = json.NewDecoder(r.Body).Decode(&job)
		r.Body.Close()
		if err != nil {
			t.Fatalf("decode job: %v", err)
		}

		if job.Status == entity.JobSucceeded || job.Status == entity.JobFailed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting job %d to finish, last status=%s", id, job.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}

	queueAfterReq := getJobsTotal(t, "queued")
	runningAfterReq := getJobsTotal(t, "running")
	succeededAfterReq := getJobsTotal(t, "succeeded")

	latencyAfterReq := expvar.Get("job_processing_latency_ns_count").(*expvar.Int).Value()

	if queueAfterReq != expectedQueue {
		t.Fatalf("queued=%d want=%d", queueAfterReq, expectedQueue)
	}

	if runningAfterReq != expectedRunning {
		t.Fatalf("queued=%d want=%d", runningAfterReq, expectedRunning)
	}

	if succeededAfterReq != expectedSucceeded {
		t.Fatalf("succeeded=%d want=%d", succeededAfterReq, expectedSucceeded)
	}

	if latencyAfterReq != expectedLatency {
		t.Fatalf("latency_count=%d want=%d", latencyAfterReq, expectedLatency)
	}

}

func getJobsTotal(t *testing.T, key string) int64 {
	t.Helper()
	v := expvar.Get("jobs_total")
	if v == nil {
		t.Fatalf("expvar jobs_total missing")
	}
	m := v.(*expvar.Map)
	vv := m.Get(key)
	if vv == nil {
		return 0
	}
	n, err := strconv.ParseInt(vv.String(), 10, 64)
	if err != nil {
		t.Fatalf("parse jobs_total[%s]=%q: %v", key, vv.String(), err)
	}
	return n
}
