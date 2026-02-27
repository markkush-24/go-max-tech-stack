package middleware

import (
	"net/http"
	"pet-study/internal/config"
	"pet-study/internal/httputils"
	"pet-study/internal/security"
	"strconv"
	"strings"
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
}

func NewCORS(cfg config.CORSConfig) *CORSAPI {
	a := &CORSAPI{
		allowedOrigins:   map[string]struct{}{},
		allowedMethods:   append([]string(nil), cfg.AllowedMethods...),
		allowedMethodsUp: map[string]struct{}{},
		allowedHeaders:   append([]string(nil), cfg.AllowedHeaders...),
		allowedHeadersLo: map[string]struct{}{},
		allowCredentials: cfg.AllowCredentials,
		maxAgeSeconds:    int(cfg.MaxAge.Seconds()),
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

		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		if c.allowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Preflight: OPTIONS + Access-Control-Request-Method
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			reqMethod := strings.ToUpper(strings.TrimSpace(r.Header.Get("Access-Control-Request-Method")))
			if !c.isMethodAllowed(reqMethod) {
				_ = httputils.WriteProblem(w, r, httputils.Problem{
					Status: http.StatusForbidden,
					Detail: "cors method denied",
				})
				return
			}

			if !c.areRequestHeadersAllowed(r.Header.Get("Access-Control-Request-Headers")) {
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

var _ = security.AuthZKind // чтобы линтер не ругался, если security используется где-то ещё
