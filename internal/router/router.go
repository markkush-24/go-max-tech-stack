package router

import (
	"net/http"
)

func NewRouter(h http.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/api/v1/users", h)
	mux.Handle("/api/v1/users/", h)

	return WithProblemNotFound(mux)
}
