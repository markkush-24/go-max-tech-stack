package httputils

import (
	"errors"
	"fmt"
	"net/http"
	"pet-study/internal/entity"
	"pet-study/internal/queue"
	"pet-study/internal/service"
	"strconv"
	"time"
)

// ValidationError — семантическая валидация (422) с invalid_params.
// Её будет возвращать хендлер после ValidateCreateUserInput (или более общий валидатор).
type ValidationError struct {
	InvalidParams []InvalidParam
}

func (e *ValidationError) Error() string { return "validation failed" }

// MethodNotAllowedError — 405 + Allow header.
type MethodNotAllowedError struct {
	Allow string
}

func (e *MethodNotAllowedError) Error() string { return "method not allowed" }

// BadRequestError — удобный typed error для “ручных” 400 (например invalid_id).
type BadRequestError struct {
	Detail string
}

func (e *BadRequestError) Error() string {
	if e.Detail == "" {
		return "bad request"
	}
	return e.Detail
}

// MappedProblem — результат централизованного маппинга.
// Headers нужны, потому что 405 требует Allow (RFC 9110).
// Log=true — для “unexpected” (500) и похожих системных ошибок.
type MappedProblem struct {
	Problem Problem
	Headers http.Header
	Log     bool
}

// MapError превращает любую ошибку в Problem+JSON + (опционально) заголовки.
// Здесь же решаем, что логировать централизованно.
//
// Правила:
// - user-facing ошибки (400/413/415/422/404/409/403/405) обычно НЕ логируем,
// - неожиданные (500) — логируем.
func MapError(r *http.Request, err error) MappedProblem {
	// Защита от typed-nil/неожиданностей.
	if err == nil {
		return MappedProblem{
			Problem: Problem{Status: http.StatusInternalServerError, Detail: "internal server error"},
			Log:     true,
		}
	}

	// 1) Если хендлер вернул “готовую” typed-ошибку HTTP.
	var bre *BadRequestError
	if errors.As(err, &bre) {
		return MappedProblem{
			Problem: Problem{Status: http.StatusBadRequest, Detail: bre.Detail},
		}
	}

	var mae *MethodNotAllowedError
	if errors.As(err, &mae) {
		h := make(http.Header, 1)
		if mae.Allow != "" {
			h.Set("Allow", mae.Allow)
		}
		return MappedProblem{
			Problem: Problem{Status: http.StatusMethodNotAllowed, Detail: "method not allowed"},
			Headers: h,
		}
	}

	var ve *ValidationError
	if errors.As(err, &ve) {
		return MappedProblem{
			Problem: Problem{
				Status:        http.StatusUnprocessableEntity,
				Detail:        "validation failed",
				InvalidParams: ve.InvalidParams,
			},
		}
	}

	// 2) Payload too large (413) — приходит как *http.MaxBytesError из MaxBytesReader.
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) {
		return MappedProblem{
			Problem: Problem{Status: http.StatusRequestEntityTooLarge, Detail: "request body too large"},
		}
	}

	// 3) Content-Type (415) — твой RequireJSONContentType заворачивает ErrUnsupportedMediaType.
	if errors.Is(err, ErrUnsupportedMediaType) {
		return MappedProblem{
			Problem: Problem{Status: http.StatusUnsupportedMediaType, Detail: "Content-Type must be application/json"},
		}
	}

	// 4) JSON decode (400) — твой ParseJSON возвращает JSONRequestError с Kind=ErrJSON...
	// Детали (field) берём через errors.As в *JSONRequestError.
	var jre *JSONRequestError
	_ = errors.As(err, &jre)

	switch {
	case errors.Is(err, ErrJSONEmptyBody):
		return MappedProblem{Problem: Problem{Status: http.StatusBadRequest, Detail: "request body must not be empty"}}

	case errors.Is(err, ErrJSONBadSyntax):
		return MappedProblem{Problem: Problem{Status: http.StatusBadRequest, Detail: "malformed JSON"}}

	case errors.Is(err, ErrJSONUnknownField):
		detail := "unknown field"
		if jre != nil && jre.Field != "" {
			detail = fmt.Sprintf("unknown field %q", jre.Field)
		}
		return MappedProblem{Problem: Problem{Status: http.StatusBadRequest, Detail: detail}}

	case errors.Is(err, ErrJSONTypeMismatch):
		detail := "invalid JSON type"
		if jre != nil && jre.Field != "" {
			detail = fmt.Sprintf("invalid JSON type for field %q", jre.Field)
		}
		return MappedProblem{Problem: Problem{Status: http.StatusBadRequest, Detail: detail}}

	case errors.Is(err, ErrJSONTrailingData):
		return MappedProblem{Problem: Problem{Status: http.StatusBadRequest, Detail: "request body must contain a single JSON value"}}

	case errors.Is(err, ErrJSONDecodeFailure):
		// Это “прочая” ошибка декодинга — оставляем 400 с общим detail.
		return MappedProblem{Problem: Problem{Status: http.StatusBadRequest, Detail: "invalid request body"}}
	}

	// 5) Domain/service ошибки.
	switch {
	case errors.Is(err, service.ErrNotFound):
		return MappedProblem{
			Problem: Problem{Status: http.StatusNotFound, Detail: "not found"},
		}

	case errors.Is(err, service.ErrConflict):
		return MappedProblem{
			Problem: Problem{Status: http.StatusConflict, Detail: "conflict"},
		}

	case errors.Is(err, service.ErrForbidden):
		return MappedProblem{
			Problem: Problem{Status: http.StatusForbidden, Detail: "forbidden"},
		}
	}

	// 5.1) Ошибки связанные с queue
	if errors.Is(err, queue.ErrQueueFull) {
		return MappedProblem{
			Problem: Problem{Status: http.StatusServiceUnavailable, Detail: "queue is full"},
		}
	}

	if errors.Is(err, queue.ErrQueueStopped) {
		return MappedProblem{
			Problem: Problem{Status: http.StatusServiceUnavailable, Detail: "queue is stopped"},
		}
	}

	if errors.Is(err, entity.ErrJobNotFound) {
		return MappedProblem{
			Problem: Problem{Status: http.StatusNotFound, Detail: "not found"},
		}
	}

	var rle *RateLimitError

	if errors.As(err, &rle) {
		d := rle.RetryAfter
		secsDur := (d + time.Second - 1) / time.Second
		secs := int64(secsDur / time.Second)
		if secs < 1 {
			secs = 1
		}
		h := make(http.Header, 1)
		h.Set("Retry-After", strconv.FormatInt(secs, 10))
		return MappedProblem{
			Problem: Problem{Status: http.StatusTooManyRequests, Detail: "too many requests"},
			Headers: h,
		}
	}

	// 6) Всё остальное — unexpected 500.
	// Здесь же центрально логируем.
	return MappedProblem{
		Problem: Problem{Status: http.StatusInternalServerError, Detail: "internal server error"},
		Log:     true,
	}
}
