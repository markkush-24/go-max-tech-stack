package middleware

import (
	"errors"
	"expvar"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"pet-study/internal/httputils"
	"pet-study/internal/security"
)

var (
	authzInitOnce  sync.Once
	forbiddenTotal *expvar.Int
)

type AuthorizeAPI struct {
	rules  map[string]security.RouteRule // key: r.Pattern
	logger *slog.Logger
}

func NewAuthorizeAPI(policy []security.RouteRule) (*AuthorizeAPI, error) {
	return NewAuthorizeAPIWithLogger(policy, defaultSecurityLogger())
}

func NewAuthorizeAPIWithLogger(policy []security.RouteRule, logger *slog.Logger) (*AuthorizeAPI, error) {
	authzInitOnce.Do(func() {
		forbiddenTotal = expvar.NewInt("authz_forbidden_total")
	})

	if len(policy) == 0 {
		return nil, errors.New("authorize api: policy is empty")
	}

	rules := make(map[string]security.RouteRule, len(policy))
	for _, rr := range policy {
		if rr.Pattern == "" {
			return nil, errors.New("authorize api: empty rule pattern")
		}
		if _, exists := rules[rr.Pattern]; exists {
			return nil, fmt.Errorf("authorize api: duplicate rule pattern %q", rr.Pattern)
		}
		rules[rr.Pattern] = rr
	}

	return &AuthorizeAPI{
		rules:  rules,
		logger: normalizeLogger(logger, logComponentHTTPSecurity),
	}, nil
}

func (a *AuthorizeAPI) Authorize(next httputils.AppHandler) httputils.AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
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
			forbiddenTotal.Add(1)
			err := security.NewForbidden(security.AuthZNoPolicyRule, nil)
			a.logAuthZDenied(r, security.AuthZNoPolicyRule)
			return err
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
				forbiddenTotal.Add(1)
				err := security.NewForbidden(security.AuthZAdminRequired, nil)
				a.logAuthZDenied(r, security.AuthZAdminRequired)
				return err
			}
			return next(w, r)

		default:
			forbiddenTotal.Add(1)
			err := security.NewForbidden(security.AuthZUnknownAccess, nil)
			a.logAuthZDenied(r, security.AuthZUnknownAccess)
			return err
		}
	}
}

func (a *AuthorizeAPI) logAuthZDenied(r *http.Request, kind security.AuthZErrorKind) {
	attrs := append(
		requestLogAttrs(r, http.StatusForbidden),
		slog.String(logFieldAuthZKind, string(kind)),
	)
	logSecurityDecision(r.Context(), a.logger, "security.authz.denied", attrs)
}
