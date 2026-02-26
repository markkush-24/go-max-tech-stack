package security

import (
	"errors"
	"fmt"
)

type AuthZErrorKind string

const (
	AuthZForbidden     AuthZErrorKind = "forbidden"
	AuthZAdminRequired AuthZErrorKind = "admin_required"
	AuthZNoPolicyRule  AuthZErrorKind = "no_policy_rule"
	AuthZUnknownAccess AuthZErrorKind = "unknown_access"
)

type ForbiddenError struct {
	Kind  AuthZErrorKind
	Cause error
}

func (e *ForbiddenError) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("forbidden: %s", e.Kind)
	}
	return fmt.Sprintf("forbidden: %s: %v", e.Kind, e.Cause)
}

func (e *ForbiddenError) Unwrap() error { return e.Cause }

func NewForbidden(kind AuthZErrorKind, cause error) error {
	return &ForbiddenError{Kind: kind, Cause: cause}
}

func AuthZKind(err error) (AuthZErrorKind, bool) {
	var e *ForbiddenError
	if !errors.As(err, &e) {
		return "", false
	}
	return e.Kind, true
}
