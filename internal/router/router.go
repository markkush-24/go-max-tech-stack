package router

import (
	"net/http"
	"pet-study/internal/routes"
)

func NewRouter(h *routes.UsersHandler) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/api/v1/users", h)
	mux.Handle("/api/v1/users/", h)

	return mux
}
