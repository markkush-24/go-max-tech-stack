package security

import (
	"errors"
	"fmt"
)

type AuthNErrorKind string

const (
	AuthNMissing     AuthNErrorKind = "missing"       // нет токена (скорее для middleware, но оставим как тип)
	AuthNInvalid     AuthNErrorKind = "invalid"       // подпись/формат/claims/alg/etc
	AuthNExpired     AuthNErrorKind = "expired"       // exp
	AuthNNotYetValid AuthNErrorKind = "not_yet_valid" // nbf
)

type AuthNError struct {
	Kind  AuthNErrorKind
	Cause error
}

func (e *AuthNError) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("authn: %s", e.Kind)
	}
	return fmt.Sprintf("authn: %s: %v", e.Kind, e.Cause)
}

func (e *AuthNError) Unwrap() error { return e.Cause }

func IsAuthNError(err error) bool {
	var e *AuthNError
	return errors.As(err, &e)
}

func AuthNKind(err error) (AuthNErrorKind, bool) {
	var e *AuthNError
	if !errors.As(err, &e) {
		return "", false
	}
	return e.Kind, true
}
