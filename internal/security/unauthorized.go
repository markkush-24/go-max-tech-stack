package security

import "fmt"

// UnauthorizedError — ошибка аутентификации (401).
// Kind нужен для метрик/диагностики (S6-B12), но наружу можно не светить детали.
type UnauthorizedError struct {
	Kind  AuthNErrorKind
	Cause error
}

func (e *UnauthorizedError) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("unauthorized: %s", e.Kind)
	}
	return fmt.Sprintf("unauthorized: %s: %v", e.Kind, e.Cause)
}

func (e *UnauthorizedError) Unwrap() error { return e.Cause }

// WWWAuthenticateValue — минимально по RFC6750.
func (e *UnauthorizedError) WWWAuthenticateValue() string {
	return "Bearer"
}

func NewUnauthorized(kind AuthNErrorKind, cause error) error {
	return &UnauthorizedError{Kind: kind, Cause: cause}
}

// UnauthorizedFromVerifyErr превращает ошибку из verifier.Verify(...) в 401.
func UnauthorizedFromVerifyErr(err error) error {
	if err == nil {
		return NewUnauthorized(AuthNInvalid, nil)
	}
	if kind, ok := AuthNKind(err); ok {
		return NewUnauthorized(kind, err)
	}
	return NewUnauthorized(AuthNInvalid, err)
}
