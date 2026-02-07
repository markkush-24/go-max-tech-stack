package router

import (
	"net/http"
	"pet-study/internal/httpapi"
	"pet-study/internal/httputils"
	"pet-study/internal/middleware"
)

func NewRouter(
	users httpapi.UsersAPI,
	usersV2 httpapi.UsersAPI,
	jobs httpapi.JobsAPI,
	limiter *middleware.RateLimitedAPI,
	bulkhead *middleware.BulkheadAPI,
) http.Handler {
	mux := http.NewServeMux()

	// v1 collection (method-specific patterns)
	mux.Handle("GET /api/v1/users", bulkhead.Bulkhead(limiter.RateLimiter(users.List)))
	mux.Handle("POST /api/v1/users", bulkhead.Bulkhead(limiter.RateLimiter(users.Create)))

	// v2 collection (method-specific patterns)
	mux.Handle("GET /api/v2/users", bulkhead.Bulkhead(limiter.RateLimiter(usersV2.List)))
	mux.Handle("POST /api/v2/users", bulkhead.Bulkhead(limiter.RateLimiter(usersV2.Create)))

	// v1 jobs item
	getJobByID := func(w http.ResponseWriter, r *http.Request) error {
		idStr := r.PathValue("id")
		id, ok := httputils.ParsePositiveInt(idStr)
		if !ok {
			return &httputils.BadRequestError{Detail: "id must be a positive integer"}
		}
		return jobs.GetByID(w, r, id)
	}
	mux.Handle("/api/v1/jobs/{id}", bulkhead.Bulkhead(limiter.RateLimiter(
		func(w http.ResponseWriter, r *http.Request) error {
			if r.Method == http.MethodGet {
				return getJobByID(w, r)
			}
			return &httputils.MethodNotAllowedError{Allow: "GET"}
		})))

	// v1 users item
	getUserByID := func(w http.ResponseWriter, r *http.Request) error {
		idStr := r.PathValue("id")
		id, ok := httputils.ParsePositiveInt(idStr)
		if !ok {
			return &httputils.BadRequestError{Detail: "id must be a positive integer"}
		}
		return users.GetByID(w, r, id)
	}
	mux.Handle("/api/v1/users/{id}", bulkhead.Bulkhead(limiter.RateLimiter(
		func(w http.ResponseWriter, r *http.Request) error {
			if r.Method == http.MethodGet {
				return getUserByID(w, r)
			}
			return &httputils.MethodNotAllowedError{Allow: "GET"}
		})))

	// 405 для коллекции
	mux.Handle("/api/v1/users", bulkhead.Bulkhead(limiter.RateLimiter(
		func(w http.ResponseWriter, r *http.Request) error {
			if r.Method == http.MethodGet {
				return users.List(w, r)
			}
			if r.Method == http.MethodPost {
				return users.Create(w, r)
			}
			return &httputils.MethodNotAllowedError{Allow: "GET, POST"}
		})))

	// 405 v2 (только на коллекцию, без subtree!)
	mux.Handle("/api/v2/users", bulkhead.Bulkhead(limiter.RateLimiter(
		func(w http.ResponseWriter, r *http.Request) error {
			return &httputils.MethodNotAllowedError{Allow: "GET, POST"}
		})))

	return WithProblemNotFound(mux)
}
