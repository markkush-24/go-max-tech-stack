package router

import (
	"net/http"
	"pet-study/internal/httpapi"
	"pet-study/internal/httputils"
	"pet-study/internal/middleware"
)

const mustBePositive = "id must be a positive integer"

func NewRouter(
	users httpapi.UsersAPI,
	usersV2 httpapi.UsersAPI,
	jobs httpapi.JobsAPI,
	usersProfile httpapi.UsersProfileAPI,
	limiter *middleware.RateLimitedAPI,
	bulkhead *middleware.BulkheadAPI,
	auth *middleware.AuthAPI,
	rbac *middleware.AuthorizeAPI,
) http.Handler {
	mux := http.NewServeMux()

	wrap := func(h httputils.AppHandler) httputils.AppHandler {
		return bulkhead.Bulkhead(
			limiter.RateLimiter(
				auth.Authenticate(
					rbac.Authorize(h),
				),
			),
		)
	}
	// v1 collection (method-specific patterns)
	mux.Handle("GET /api/v1/users", wrap(users.List))
	mux.Handle("POST /api/v1/users", wrap(users.Create))

	// v2 collection (method-specific patterns)
	mux.Handle("GET /api/v2/users", wrap(usersV2.List))
	mux.Handle("POST /api/v2/users", wrap(usersV2.Create))

	// v1 jobs item
	getJobByID := func(w http.ResponseWriter, r *http.Request) error {
		idStr := r.PathValue("id")
		id, ok := httputils.ParsePositiveInt(idStr)
		if !ok {
			return &httputils.BadRequestError{Detail: mustBePositive}
		}
		return jobs.GetByID(w, r, id)
	}
	mux.Handle("/api/v1/jobs/{id}", wrap(func(w http.ResponseWriter, r *http.Request) error {
		if r.Method == http.MethodGet {
			return getJobByID(w, r)
		}
		return &httputils.MethodNotAllowedError{Allow: "GET"}
	}))

	publishEvents := func(w http.ResponseWriter, r *http.Request) error {

		idStr := r.PathValue("id")
		id, ok := httputils.ParsePositiveInt(idStr)
		if !ok {
			return &httputils.BadRequestError{Detail: mustBePositive}
		}
		return jobs.Events(w, r, id)
	}
	mux.Handle("/api/v1/jobs/{id}/events", wrap(func(w http.ResponseWriter, r *http.Request) error {
		if r.Method == http.MethodGet {
			return publishEvents(w, r)
		}
		return &httputils.MethodNotAllowedError{Allow: "GET"}
	}))

	// v1 users item
	getUserByID := func(w http.ResponseWriter, r *http.Request) error {
		idStr := r.PathValue("id")
		id, ok := httputils.ParsePositiveInt(idStr)
		if !ok {
			return &httputils.BadRequestError{Detail: mustBePositive}
		}
		return users.GetByID(w, r, id)
	}
	mux.Handle("/api/v1/users/{id}", wrap(func(w http.ResponseWriter, r *http.Request) error {
		if r.Method == http.MethodGet {
			return getUserByID(w, r)
		}
		return &httputils.MethodNotAllowedError{Allow: "GET"}
	}))

	// users+profile GET
	if usersProfile != nil {
		getUserProfile := func(w http.ResponseWriter, r *http.Request) error {
			idStr := r.PathValue("id")
			id, ok := httputils.ParsePositiveInt(idStr)
			if !ok {
				return &httputils.BadRequestError{Detail: mustBePositive}
			}
			return usersProfile.GetUserProfile(w, r, int64(id))
		}
		mux.Handle("/api/v1/users/{id}/profile", wrap(func(w http.ResponseWriter, r *http.Request) error {
			if r.Method == http.MethodGet {
				return getUserProfile(w, r)
			}
			return &httputils.MethodNotAllowedError{Allow: "GET"}
		}))
	}

	// 405 для коллекции
	mux.Handle("/api/v1/users", wrap(
		func(w http.ResponseWriter, r *http.Request) error {
			if r.Method == http.MethodGet {
				return users.List(w, r)
			}
			if r.Method == http.MethodPost {
				return users.Create(w, r)
			}
			return &httputils.MethodNotAllowedError{Allow: "GET, POST"}
		}))

	// 405 v2 (только на коллекцию, без subtree!)
	mux.Handle("/api/v2/users", wrap(func(w http.ResponseWriter, r *http.Request) error {
		return &httputils.MethodNotAllowedError{Allow: "GET, POST"}
	}))

	return WithProblemNotFound(mux)
}
