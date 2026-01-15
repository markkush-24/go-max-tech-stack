package router

import (
	"net/http"
	"pet-study/internal/health"
	"time"
)

func NewHealthRouter(readiness *health.Readiness) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /livez", health.LivenessHandler())
	mux.Handle("GET /readyz", health.ReadinessHandler(readiness, 200*time.Millisecond))
	return mux
}
