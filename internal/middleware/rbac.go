package middleware

import (
	"fmt"
	"net/http"

	"pet-study/internal/httputils"
	"pet-study/internal/security"
)

type AuthorizeAPI struct {
	rules map[string]security.RouteRule // key: r.Pattern
}

func NewAuthorizeAPI(policy []security.RouteRule) *AuthorizeAPI {
	if len(policy) == 0 {
		panic("AuthorizeAPI: policy is empty")
	}

	rules := make(map[string]security.RouteRule, len(policy))
	for _, rr := range policy {
		if rr.Pattern == "" {
			panic("AuthorizeAPI: empty rule.Pattern")
		}
		if _, exists := rules[rr.Pattern]; exists {
			panic(fmt.Sprintf("AuthorizeAPI: duplicate rule.Pattern %q", rr.Pattern))
		}
		rules[rr.Pattern] = rr
	}

	return &AuthorizeAPI{rules: rules}
}

func (a *AuthorizeAPI) Authorize(next httputils.AppHandler) httputils.AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		// preflight пропускаем (CORS short-circuit будет в S6-B8)
		if r.Method == http.MethodOptions {
			return next(w, r)
		}

		pat := r.Pattern
		if pat == "" {
			// это почти всегда означает, что middleware повесили "снаружи" ServeMux
			// (или не тем способом). Лучше не превращать это в "дырку".
			return fmt.Errorf("authorize: empty r.Pattern (Authorize must run after ServeMux)")
		}

		rule, ok := a.rules[pat]
		if !ok {
			// fail-closed: забыли занести маршрут в policy => закрываем доступ
			return security.NewForbidden(security.AuthZNoPolicyRule, nil)
		}

		switch rule.Access {
		case security.AccessPublic:
			return next(w, r)

		case security.AccessAuthenticated:
			if _, ok := security.FromContext(r.Context()); !ok {
				// wiring bug: Authenticate должен быть раньше
				return fmt.Errorf("authorize: principal missing (Authenticate must run before Authorize)")
			}
			return next(w, r)

		case security.AccessAdminOnly:
			p, ok := security.FromContext(r.Context())
			if !ok {
				return fmt.Errorf("authorize: principal missing (Authenticate must run before Authorize)")
			}
			if p.Role != security.RoleAdmin {
				return security.NewForbidden(security.AuthZAdminRequired, nil)
			}
			return next(w, r)

		default:
			return security.NewForbidden(security.AuthZUnknownAccess, nil)
		}
	}
}
