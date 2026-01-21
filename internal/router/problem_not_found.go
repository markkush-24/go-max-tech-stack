package router

import (
	"net/http"
	"pet-study/internal/httputils"
)

func WithProblemNotFound(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pattern := mux.Handler(r)
		if pattern == "" {
			_ = httputils.WriteProblem(w, r, httputils.Problem{
				Status: http.StatusNotFound,
				Detail: "not found",
			})
			return
		}

		mux.ServeHTTP(w, r)
	})
}
