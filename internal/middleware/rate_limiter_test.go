package middleware_test

import (
	"mime"
	"net/http"
	"net/http/httptest"
	"pet-study/internal/metrics"
	"pet-study/internal/middleware"
	"pet-study/internal/queue"
	"pet-study/internal/requestid"
	"pet-study/internal/router"
	"pet-study/internal/routes"
	"pet-study/internal/service"
	"pet-study/internal/store/jobrepo"
	"pet-study/internal/store/userrepo"
	"strconv"
	"testing"
)

func TestRateLimiterRetryAfter(t *testing.T) {
	userRepo := userrepo.NewMemoryUserRepository()
	jobRepo := jobrepo.NewMemoryJobRepository()

	userSvc := service.NewUserService(userRepo)
	jobSvc := service.NewJobService(jobRepo)

	q := queue.New(1)
	m := metrics.DefaultHTTP()

	v1 := routes.NewUserHandler(userSvc, jobSvc, q, m)
	v2 := routes.NewUserV2Handler(userSvc, jobSvc, q, m)

	lim := middleware.NewRateLimitedAPI(float64(1), 1)
	bh := middleware.NewBulkhead(1)

	jh := routes.NewJobHandler(jobSvc)

	userRouter := router.NewRouter(v1, v2, jh, lim, bh)

	rootRouter := router.NewRoot(userRouter, http.NewServeMux(), nil)

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

	resp, err := server.Client().Get(server.URL + "/api/v1/users")
	if err != nil {
		t.Fatalf("get #1: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get #1 status=%d want=%d", resp.StatusCode, http.StatusOK)
	}

	resp2, err2 := server.Client().Get(server.URL + "/api/v1/users")
	if err2 != nil {
		t.Fatalf("get #2: %v", err2)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("get #2 status=%d want=%d", resp2.StatusCode, http.StatusTooManyRequests)
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

	retryAfter2 := resp2.Header.Get("Retry-After")
	if retryAfter2 == "" {
		t.Fatalf("%s empty", "Retry-After")
	}
	n, err := strconv.Atoi(retryAfter2)
	if err != nil {
		t.Fatalf("%v convert error", err)
	}
	if n < 1 {
		t.Fatalf("Retry-After must be >= 1 , current %v", n)
	}
}
