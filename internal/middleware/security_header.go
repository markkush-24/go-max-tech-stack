package middleware

import (
	"net/http"
	"strconv"

	"pet-study/internal/config"
)

type SecurityHeadersAPI struct {
	cfg config.SecurityHeadersConfig
}

func NewSecurityHeaders(cfg config.SecurityHeadersConfig) *SecurityHeadersAPI {
	return &SecurityHeadersAPI{cfg: cfg}
}

func (s *SecurityHeadersAPI) SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Enable {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("X-Frame-Options", "DENY")

			// HSTS: only on HTTPS
			if s.cfg.HSTS.MaxAge > 0 && r.TLS != nil {
				w.Header().Set("Strict-Transport-Security", "max-age="+strconv.Itoa(int(s.cfg.HSTS.MaxAge)))
			}
		}

		next.ServeHTTP(w, r)
	})
}
