package middleware

import (
	"net/http"
	"pet-study/internal/security"
	"strconv"
	"time"

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
			if s.cfg.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", s.cfg.ReferrerPolicy)
			}
			w.Header().Set("X-Frame-Options", "DENY")

			// HSTS: only on HTTPS
			if s.cfg.HSTS.Enable && s.cfg.HSTS.MaxAge > 0 {
				isHTTPS := false

				if ri, ok := security.RequestInfoFromContext(r.Context()); ok {
					isHTTPS = ri.Scheme == "https"
				} else {
					isHTTPS = r.TLS != nil
				}

				if isHTTPS {
					secs := int64(s.cfg.HSTS.MaxAge / time.Second)
					w.Header().Set("Strict-Transport-Security", "max-age="+strconv.FormatInt(secs, 10))
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}
