package router

import (
	"net/http"
	"pet-study/internal/health"
	"pet-study/internal/httputils"
)

func NewHealthRouter(readiness *health.Readiness) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		_ = httputils.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !readiness.IsReady() {
			_ = httputils.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
			return
		}
		_ = httputils.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return mux
}
