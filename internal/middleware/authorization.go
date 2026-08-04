package middleware

import (
	"errors"
	"expvar"
	"log/slog"
	"net/http"
	"pet-study/internal/httputils"
	"pet-study/internal/security"
	"strings"
	"sync"
)

var (
	authnInitOnce sync.Once
	authnFailures *expvar.Map
)

type AuthAPI struct {
	verifier security.Verifier
	logger   *slog.Logger
}

func NewAuthAPI(verifier security.Verifier) (*AuthAPI, error) {
	return NewAuthAPIWithLogger(verifier, defaultSecurityLogger())
}

func NewAuthAPIWithLogger(verifier security.Verifier, logger *slog.Logger) (*AuthAPI, error) {
	authnInitOnce.Do(func() {
		authnFailures = expvar.NewMap("authn_failures_total")
	})

	if verifier == nil {
		return nil, errors.New("auth api: verifier is nil")
	}
	return &AuthAPI{
		verifier: verifier,
		logger:   normalizeLogger(logger, logComponentHTTPSecurity),
	}, nil
}

func (a *AuthAPI) Authenticate(next httputils.AppHandler) httputils.AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if r.Method == http.MethodOptions {
			return next(w, r)
		}

		token, err := parseBearerToken(r.Header.Get("Authorization"))
		if err != nil {
			k := authNKindOrOther(err)
			incAuthN(k)
			a.logAuthNDenied(r, k)
			return err
		}

		p, err := a.verifier.Verify(token)
		if err != nil {
			uerr := security.UnauthorizedFromVerifyErr(err)
			k := authNKindOrOther(uerr)
			incAuthN(k)
			a.logAuthNDenied(r, k)
			return uerr
		}

		r = r.WithContext(security.WithPrincipal(r.Context(), p))
		return next(w, r)
	}
}

func parseBearerToken(hdr string) (string, error) {
	hdr = strings.TrimSpace(hdr)
	if hdr == "" {
		return "", security.NewUnauthorized(security.AuthNMissing, nil)
	}

	parts := strings.Fields(hdr)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", security.NewUnauthorized(security.AuthNInvalid, nil)
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", security.NewUnauthorized(security.AuthNInvalid, nil)
	}
	return token, nil
}

func (a *AuthAPI) logAuthNDenied(r *http.Request, kind security.AuthNErrorKind) {
	attrs := append(
		requestLogAttrs(r, http.StatusUnauthorized),
		slog.String(logFieldAuthNKind, string(kind)),
	)
	logSecurityDecision(r.Context(), a.logger, "security.authn.denied", attrs)
}

func authNKindOrOther(err error) security.AuthNErrorKind {
	if k, ok := security.AuthNKind(err); ok {
		return k
	}
	return "other"
}

func incAuthN(kind security.AuthNErrorKind) {
	key := string(kind)
	v := authnFailures.Get(key)
	if v == nil {
		n := new(expvar.Int)
		authnFailures.Set(key, n)
		v = n
	}
	v.(*expvar.Int).Add(1)
}
