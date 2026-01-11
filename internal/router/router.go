package router

import (
	"net/http"
	"pet-study/internal/httpapi"
	"pet-study/internal/httputils"
)

func NewRouter(users httpapi.UsersAPI, usersV2 httpapi.UsersAPI) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /api/v1/users", httputils.AppHandler(users.List))
	mux.Handle("POST /api/v1/users", httputils.AppHandler(users.Create))

	mux.Handle("GET /api/v2/users", httputils.AppHandler(usersV2.List))
	mux.Handle("POST /api/v2/users", httputils.AppHandler(usersV2.Create))

	getByID := func(w http.ResponseWriter, r *http.Request) error {
		idStr := r.PathValue("id")
		id, ok := httputils.ParsePositiveInt(idStr)
		if !ok {
			return &httputils.BadRequestError{Detail: "id must be a positive integer"}
		}
		return users.GetByID(w, r, id)
	}

	mux.Handle("GET /api/v1/users/{id}", httputils.AppHandler(getByID))

	// 405 для item: матчим только ровно один сегмент {id}, а не всё под /users/
	mux.Handle("/api/v1/users/{id}", httputils.AppHandler(func(w http.ResponseWriter, r *http.Request) error {
		if r.Method == http.MethodGet {
			return getByID(w, r)
		}
		return &httputils.MethodNotAllowedError{Allow: "GET"}
	}))

	// 405 для коллекции
	mux.Handle("/api/v1/users", httputils.AppHandler(func(w http.ResponseWriter, r *http.Request) error {
		if r.Method == http.MethodGet {
			return users.List(w, r)
		}
		if r.Method == http.MethodPost {
			return users.Create(w, r)
		}
		return &httputils.MethodNotAllowedError{Allow: "GET, POST"}
	}))

	// 405 v2 (только на коллекцию, без subtree!)
	mux.Handle("/api/v2/users", httputils.AppHandler(func(w http.ResponseWriter, r *http.Request) error {
		return &httputils.MethodNotAllowedError{Allow: "GET, POST"}
	}))

	return WithProblemNotFound(mux)
}
