package httputils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strings"
)

var rxUnknownField = regexp.MustCompile(`^json: unknown field "([^"]+)"$`) // эвристика: stdlib не даёт typed-ошибку

func ParseJSON(r io.Reader, dst any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	// 1) Первый decode
	if err := dec.Decode(dst); err != nil {
		// empty body
		if errors.Is(err, io.EOF) {
			return &JSONRequestError{Kind: ErrJSONEmptyBody, Cause: err}
		}

		// syntax error (битый JSON)
		var se *json.SyntaxError
		if errors.As(err, &se) {
			return &JSONRequestError{Kind: ErrJSONBadSyntax, Offset: se.Offset, Cause: err}
		}

		if errors.Is(err, io.ErrUnexpectedEOF) {
			return &JSONRequestError{Kind: ErrJSONBadSyntax, Cause: err}
		}

		// type mismatch (например, ожидаем string, пришло число)
		var ute *json.UnmarshalTypeError
		if errors.As(err, &ute) {
			return &JSONRequestError{Kind: ErrJSONTypeMismatch, Field: ute.Field, Cause: err}
		}

		// unknown field (stdlib возвращает только текст, поэтому парсим текст — это эвристика)
		if m := rxUnknownField.FindStringSubmatch(err.Error()); len(m) == 2 {
			return &JSONRequestError{Kind: ErrJSONUnknownField, Field: m[1], Cause: err}
		}

		// всё остальное (включая неожиданные кейсы)
		return &JSONRequestError{Kind: ErrJSONDecodeFailure, Cause: err}
	}

	// 2) Проверяем, что больше НИЧЕГО нет (второй JSON / мусор)
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		// err может быть nil (если там валидный второй JSON)
		// или SyntaxError (если мусор после первого JSON)
		return &JSONRequestError{Kind: ErrJSONTrailingData, Cause: err}
	}

	return nil
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	var buf bytes.Buffer

	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if _, err := w.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("write response: %w", err)
	}
	return nil
}

func RequireJSONContentType(r *http.Request) error {
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		return fmt.Errorf("%w: missing Content-Type", ErrUnsupportedMediaType)
	}

	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return fmt.Errorf("%w: invalid Content-Type", ErrUnsupportedMediaType)
	}

	if !strings.EqualFold(mediaType, "application/json") {
		return fmt.Errorf("%w: %s", ErrUnsupportedMediaType, mediaType)
	}

	return nil
}

func LimitBody(w http.ResponseWriter, r *http.Request, max int64) {
	r.Body = http.MaxBytesReader(w, r.Body, max)
}

type ErrorDetail struct {
	Field string `json:"field,omitempty"`
	Rule  string `json:"rule,omitempty"`
}

type ErrorBody struct {
	Error struct {
		Code    string        `json:"code"`
		Message string        `json:"message"`
		Details []ErrorDetail `json:"details,omitempty"`
	} `json:"error"`
}

func WriteError(w http.ResponseWriter, status int, code, message string, details ...ErrorDetail) error {
	var body ErrorBody
	body.Error.Code = code
	body.Error.Message = message
	body.Error.Details = details
	return WriteJSON(w, status, body)
}
