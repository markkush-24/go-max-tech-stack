package router

import (
	"net/http"
	"pet-study/internal/httputils"
)

func WithProblemNotFound(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, pattern := mux.Handler(r)
		if pattern == "" {
			_ = httputils.WriteProblem(w, r, httputils.Problem{
				Status: http.StatusNotFound,
				Detail: "not found",
			})
			return
		}
		h.ServeHTTP(w, r)
	})
}
