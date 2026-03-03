package middleware

import (
	"net/http"
	"pet-study/internal/requestid"
)

func (p *ProxyAPI) SanitizeRequestIDHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteIP := parseRemoteIP(r.RemoteAddr)
		if remoteIP == "" || !p.isTrustedProxy(remoteIP) {
			r.Header.Del(requestid.HeaderName)
		}
		next.ServeHTTP(w, r)
	})
}
