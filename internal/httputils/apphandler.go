package httputils

import (
	"log/slog"
	"net/http"
	"pet-study/internal/requestid"
)

type AppHandler func(w http.ResponseWriter, r *http.Request) error

func (h AppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		mp := MapError(r, err)

		for k, vv := range mp.Headers {
			w.Header().Del(k)
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}

		if mp.Log {
			rid, _ := requestid.RequestID(r.Context())
			slog.Default().With("component", "app_handler").Error(
				"request failed",
				"method", r.Method,
				"path", r.URL.Path,
				"request_id", rid,
				"err", err,
			)
		}

		_ = WriteProblem(w, r, mp.Problem)
	}
}
