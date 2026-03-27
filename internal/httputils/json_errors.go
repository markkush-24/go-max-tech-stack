package httputils

import (
	"errors"
	"fmt"
)

// Sentinel-ошибки: по ним удобно делать errors.Is(...)
var (
	ErrJSONEmptyBody        = errors.New("json: empty body")
	ErrJSONTrailingData     = errors.New("json: trailing data")
	ErrJSONUnknownField     = errors.New("json: unknown field")
	ErrJSONBadSyntax        = errors.New("json: bad syntax")
	ErrJSONTypeMismatch     = errors.New("json: type mismatch")
	ErrJSONDecodeFailure    = errors.New("json: decode failure")
	ErrUnsupportedMediaType = errors.New("unsupported media type")
	ErrStreamingUnsupported = errors.New("streaming is not supported")
)

// JSONRequestError — “обёртка” вокруг причин; даёт и sentinel, и оригинальную причину.
type JSONRequestError struct {
	Kind   error  // один из ErrJSON
	Field  string // для unknown field / type mismatch (если применимо)
	Offset int64  // для syntax error (если применимо)
	Cause  error  // оригинальная ошибка декодера
}

func (e *JSONRequestError) Error() string {
	switch {
	case e.Field != "":
		return fmt.Sprintf("%v: field=%q: %v", e.Kind, e.Field, e.Cause)
	case e.Offset != 0:
		return fmt.Sprintf("%v: offset=%d: %v", e.Kind, e.Offset, e.Cause)
	case e.Cause != nil:
		return fmt.Sprintf("%v: %v", e.Kind, e.Cause)
	default:
		return e.Kind.Error()
	}
}

// Unwrap() []error позволяет errors.Is/As видеть и Kind, и Cause.
func (e *JSONRequestError) Unwrap() []error {
	if e.Cause == nil {
		return []error{e.Kind}
	}
	return []error{e.Kind, e.Cause}
}
