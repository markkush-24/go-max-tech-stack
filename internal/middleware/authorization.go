package middleware

import (
	"net/http"
	"pet-study/internal/httputils"
	"pet-study/internal/security"
	"strings"
)

type AuthAPI struct {
	verifier security.Verifier
}

func NewAuthAPI(verifier security.Verifier) *AuthAPI {
	if verifier == nil {
		panic("AuthAPI: verifier is nil")
	}
	return &AuthAPI{verifier: verifier}
}

func (a *AuthAPI) Authenticate(next httputils.AppHandler) httputils.AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if r.Method == http.MethodOptions {
			return next(w, r)
		}

		token, err := parseBearerToken(r.Header.Get("Authorization"))
		if err != nil {
			return err
		}

		p, err := a.verifier.Verify(token)
		if err != nil {
			return security.UnauthorizedFromVerifyErr(err)
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
