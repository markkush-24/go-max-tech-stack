package middleware

import (
	"net"
	"net/http"
	"pet-study/internal/security"
	"strings"

	"pet-study/internal/config"
)

type ProxyAPI struct {
	trustedNets []*net.IPNet
	trustXFF    bool
	trustXFP    bool
}

func NewProxyAPI(cfg config.ProxyConfig) (*ProxyAPI, error) {
	// Предполагаю, что в cfg.Proxy.TrustedProxies лежит []string (CIDR или IP).
	// Если поле у тебя называется иначе — подгони здесь.
	nets := make([]*net.IPNet, 0, len(cfg.TrustedProxies))
	for _, s := range cfg.TrustedProxies {
		space := strings.TrimSpace(s.String())
		if space == "" {
			continue
		}

		// CIDR?
		if strings.Contains(space, "/") {
			_, ipnet, err := net.ParseCIDR(space)
			if err != nil {
				return nil, err
			}
			nets = append(nets, ipnet)
			continue
		}

		// Plain IP -> /32 or /128
		ip := net.ParseIP(space)
		if ip == nil {
			return nil, &net.AddrError{Err: "invalid ip", Addr: space}
		}
		if ip.To4() != nil {
			_, ipnet, _ := net.ParseCIDR(space + "/32")
			nets = append(nets, ipnet)
		} else {
			_, ipnet, _ := net.ParseCIDR(space + "/128")
			nets = append(nets, ipnet)
		}
	}

	return &ProxyAPI{
		trustedNets: nets,
		trustXFF:    cfg.TrustXFF,
		trustXFP:    cfg.TrustXFP,
	}, nil
}

func (p *ProxyAPI) TrustProxy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteIP := parseRemoteIP(r.RemoteAddr)

		// defaults (direct-connect truth)
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		clientIP := remoteIP

		if remoteIP != "" && p.isTrustedProxy(remoteIP) {
			if p.trustXFP {
				if xfp := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))); xfp == "http" || xfp == "https" {
					scheme = xfp
				}
			}

			if p.trustXFF {
				if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
					if ip := firstIPFromXFF(xff); ip != "" {
						clientIP = ip
					}
				}
			}
		}

		ri := security.RequestInfo{
			ClientIP: clientIP,
			Scheme:   scheme,
		}
		r = r.WithContext(security.WithRequestInfo(r.Context(), ri))
		next.ServeHTTP(w, r)
	})
}

func (p *ProxyAPI) isTrustedProxy(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, n := range p.trustedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func parseRemoteIP(remoteAddr string) string {
	if remoteAddr == "" {
		return ""
	}
	// usually "ip:port"
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}
	// maybe plain ip without port
	if net.ParseIP(remoteAddr) != nil {
		return remoteAddr
	}
	return ""
}

func firstIPFromXFF(xff string) string {
	// стандартная семантика XFF: самый левый — “похоже на исходного клиента”,
	// когда прокси добавляют себя справа.
	parts := strings.Split(xff, ",")
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		// XFF может содержать ip:port (редко) — поддержим
		if host, _, err := net.SplitHostPort(s); err == nil {
			s = host
		}
		if net.ParseIP(s) != nil {
			return s
		}
		// если первый элемент не IP — пробуем следующий
	}
	return ""
}
