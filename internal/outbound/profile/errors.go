package profile

import (
	"errors"
	"fmt"
)

var (
	ErrTimeout     = errors.New("timeout")
	ErrCanceled    = errors.New("canceled")
	ErrNetwork     = errors.New("network")
	ErrUpstream5xx = errors.New("upstream 5xx")
	ErrUpstream4xx = errors.New("upstream 4xx")
	ErrParse       = errors.New("parse")
	ErrBadResponse = errors.New("bad response")
)

type Error struct {
	Kind   error
	Status int
	Cause  error
}

func (e *Error) Error() string {
	if e.Status != 0 && e.Cause != nil {
		return fmt.Sprintf("profile %v: status=%d: %v", e.Kind, e.Status, e.Cause)
	}
	if e.Status != 0 {
		return fmt.Sprintf("profile %v: status=%d", e.Kind, e.Status)
	}
	if e.Cause != nil {
		return fmt.Sprintf("profile %v: %v", e.Kind, e.Cause)
	}
	return fmt.Sprintf("profile %v", e.Kind)
}

func (e *Error) Unwrap() []error {
	if e.Cause == nil {
		return []error{e.Kind}
	}
	return []error{e.Kind, e.Cause}
}

func KindLabel(err error) string {
	switch {
	case errors.Is(err, ErrTimeout):
		return "timeout"
	case errors.Is(err, ErrCanceled):
		return "canceled"
	case errors.Is(err, ErrNetwork):
		return "network"
	case errors.Is(err, ErrUpstream5xx):
		return "5xx"
	case errors.Is(err, ErrUpstream4xx):
		return "4xx"
	case errors.Is(err, ErrParse):
		return "parse"
	case errors.Is(err, ErrBadResponse):
		return "bad_response"
	default:
		return "unknown"
	}
}
