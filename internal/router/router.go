package router

import (
	"net/http"
	"pet-study/internal/middleware"
	"pet-study/internal/routes"
)

func NewRouter(h *routes.UsersHandler) http.Handler {
	mux := http.NewServeMux()

	wrapped := middleware.MiddleWareLogger(middleware.MiddleWareRecover(h))
	mux.Handle("/api/v1/users", wrapped)
	mux.Handle("/api/v1/users/", wrapped)

	return mux
}
