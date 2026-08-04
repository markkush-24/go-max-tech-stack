package middleware

import (
	"expvar"
	"log/slog"
	"net/http"
	"pet-study/internal/config"
	"pet-study/internal/httputils"
	"strconv"
	"strings"
	"sync"
)

var (
	corsInitOnce       sync.Once
	corsPreflightTotal *expvar.Int
	corsDeniedTotal    *expvar.Int
)

type CORSAPI struct {
	allowAny         bool
	allowedOrigins   map[string]struct{} // exact match
	allowedMethods   []string
	allowedMethodsUp map[string]struct{} // upper
	allowedHeaders   []string
	allowedHeadersLo map[string]struct{} // lower
	allowCredentials bool
	maxAgeSeconds    int
	logger           *slog.Logger
}

func NewCORS(cfg config.CORSConfig) *CORSAPI {
	return NewCORSWithLogger(cfg, defaultSecurityLogger())
}

func NewCORSWithLogger(cfg config.CORSConfig, logger *slog.Logger) *CORSAPI {
	corsInitOnce.Do(func() {
		corsPreflightTotal = expvar.NewInt("cors_preflight_total")
		corsDeniedTotal = expvar.NewInt("cors_denied_total")
	})

	a := &CORSAPI{
		allowedOrigins:   map[string]struct{}{},
		allowedMethods:   append([]string(nil), cfg.AllowedMethods...),
		allowedMethodsUp: map[string]struct{}{},
		allowedHeaders:   append([]string(nil), cfg.AllowedHeaders...),
		allowedHeadersLo: map[string]struct{}{},
		allowCredentials: cfg.AllowCredentials,
		maxAgeSeconds:    int(cfg.MaxAge.Seconds()),
		logger:           normalizeLogger(logger, logComponentHTTPSecurity),
	}

	// origins
	for _, o := range cfg.AllowedOrigins {
		if o == "*" {
			a.allowAny = true
			continue
		}
		a.allowedOrigins[o] = struct{}{}
	}

	// methods
	for _, m := range a.allowedMethods {
		m = strings.ToUpper(strings.TrimSpace(m))
		if m == "" {
			continue
		}
		a.allowedMethodsUp[m] = struct{}{}
	}

	// headers
	for _, h := range a.allowedHeaders {
		h = strings.ToLower(strings.TrimSpace(h))
		if h == "" {
			continue
		}
		a.allowedHeadersLo[h] = struct{}{}
	}

	return a
}

// CORS — http.Handler middleware, применяется к API-роутеру (а не к /livez, /readyz, /debug/*).
// Политика: deny-by-default для запросов с Origin.
func (c *CORSAPI) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// не CORS-сценарий (или не браузер) — ничего не делаем
			next.ServeHTTP(w, r)
			return
		}

		if !c.isOriginAllowed(origin) {
			corsDeniedTotal.Add(1)
			c.logCORSDenied(r, "origin")
			_ = httputils.WriteProblem(w, r, httputils.Problem{
				Status: http.StatusForbidden,
				Detail: "cors origin denied",
			})
			return
		}

		allowOrigin := origin
		if c.allowAny {
			allowOrigin = "*"
		} else {
			// важное для кешей
			httputils.AddVary(w, "Origin")
		}

		if c.allowCredentials && allowOrigin == "*" {
			// в этом режиме браузер всё равно откажет, лучше явно deny
			corsDeniedTotal.Add(1)
			c.logCORSDenied(r, "credentials_wildcard")
			_ = httputils.WriteProblem(w, r, httputils.Problem{
				Status: http.StatusForbidden,
				Detail: "cors credentials with wildcard origin is not allowed",
			})
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		if c.allowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Preflight: OPTIONS + Access-Control-Request-Method
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			reqMethod := strings.ToUpper(strings.TrimSpace(r.Header.Get("Access-Control-Request-Method")))
			if !c.isMethodAllowed(reqMethod) {
				corsDeniedTotal.Add(1)
				c.logCORSDenied(r, "method")
				_ = httputils.WriteProblem(w, r, httputils.Problem{
					Status: http.StatusForbidden,
					Detail: "cors method denied",
				})
				return
			}

			if !c.areRequestHeadersAllowed(r.Header.Get("Access-Control-Request-Headers")) {
				corsDeniedTotal.Add(1)
				c.logCORSDenied(r, "headers")
				_ = httputils.WriteProblem(w, r, httputils.Problem{
					Status: http.StatusForbidden,
					Detail: "cors headers denied",
				})
				return
			}

			w.Header().Set("Access-Control-Allow-Methods", strings.Join(c.allowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(c.allowedHeaders, ", "))

			if c.maxAgeSeconds > 0 {
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(c.maxAgeSeconds))
			}

			// для корректного кеширования preflight
			httputils.AddVary(w, "Access-Control-Request-Method")
			httputils.AddVary(w, "Access-Control-Request-Headers")

			corsPreflightTotal.Add(1)
			w.WriteHeader(http.StatusNoContent) // 204
			return
		}

		// обычный запрос
		next.ServeHTTP(w, r)
	})
}

func (c *CORSAPI) isOriginAllowed(origin string) bool {
	if c.allowAny {
		return true
	}
	_, ok := c.allowedOrigins[origin]
	return ok
}

func (c *CORSAPI) isMethodAllowed(m string) bool {
	_, ok := c.allowedMethodsUp[m]
	return ok
}

func (c *CORSAPI) areRequestHeadersAllowed(acrh string) bool {
	acrh = strings.TrimSpace(acrh)
	if acrh == "" {
		return true
	}
	parts := strings.Split(acrh, ",")
	for _, p := range parts {
		h := strings.ToLower(strings.TrimSpace(p))
		if h == "" {
			continue
		}
		if _, ok := c.allowedHeadersLo[h]; !ok {
			return false
		}
	}
	return true
}

func (c *CORSAPI) logCORSDenied(r *http.Request, reason string) {
	attrs := append(
		requestLogAttrs(r, http.StatusForbidden),
		slog.String(logFieldCORSDenial, reason),
	)
	logSecurityDecision(r.Context(), c.logger, "security.cors.denied", attrs)
}
