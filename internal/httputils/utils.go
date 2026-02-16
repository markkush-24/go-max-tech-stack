package httputils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"pet-study/internal/entity"
	"regexp"
	"strings"
)

var rxUnknownField = regexp.MustCompile(`^json: unknown field "([^"]+)"$`) // эвристика: stdlib не даёт typed-ошибку
var ErrNotImplemented = errors.New("not implemented yet")

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

func WriteNegotiated(w http.ResponseWriter, r *http.Request, status int, v any, msg any) error {
	ct, err := AcceptHeader(r.Header.Get("Accept"))
	if err != nil {
		return err // в errmap -> 406
	}
	AddVary(w, "Accept")

	switch ct {
	case MediaTypeProtobuf:
		return WriteProtobuf(w, status, msg)

	case MediaTypeJSON:
		return WriteJSON(w, status, v)
	default:
		return ErrNotAcceptable
	}
}

func WriteProtobuf(w http.ResponseWriter, status int, msg any) error {
	return ErrNotImplemented
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

func ValidateCreateUserInput(in entity.CreateUserInput) []ErrorDetail {
	var ed []ErrorDetail

	if in.Age < 0 {
		ed = append(ed, ErrorDetail{Field: "age", Rule: "must be >= 0"})
	}

	if strings.TrimSpace(in.Name) == "" {
		ed = append(ed, ErrorDetail{Field: "name", Rule: "must not be empty"})
	}

	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == "" {
		ed = append(ed, ErrorDetail{Field: "email", Rule: "must not be empty"})
	} else if !isLikelyEmail(email) {
		ed = append(ed, ErrorDetail{Field: "email", Rule: "email must be valid"})
	}

	return ed
}

func ValidateCreateUserInputV2(in entity.CreateUserInputV2) []ErrorDetail {
	var ed []ErrorDetail

	if in.Age < 0 {
		ed = append(ed, ErrorDetail{Field: "age", Rule: "must be >= 0"})
	}

	if strings.TrimSpace(in.FullName) == "" {
		ed = append(ed, ErrorDetail{Field: "full_name", Rule: "must not be empty"})
	}

	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == "" {
		ed = append(ed, ErrorDetail{Field: "email", Rule: "must not be empty"})
	} else if !isLikelyEmail(email) {
		ed = append(ed, ErrorDetail{Field: "email", Rule: "email must be valid"})
	}

	return ed
}

func isLikelyEmail(s string) bool {
	// ожидаем уже TrimSpace + ToLower снаружи, но можно и тут
	if s == "" {
		return false
	}

	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 { // '@' не может быть первым или последним
		return false
	}

	// должен быть ровно один '@'
	if strings.IndexByte(s[at+1:], '@') != -1 {
		return false
	}

	local := s[:at]
	domain := s[at+1:]

	// local и domain не пустые уже гарантированы at-check'ом, но оставим явно
	if local == "" || domain == "" {
		return false
	}

	// минимальная проверка домена: хотя бы одна точка и не в начале/конце домена
	dot := strings.LastIndexByte(domain, '.')
	if dot <= 0 || dot == len(domain)-1 {
		return false
	}

	return true
}

func AddVary(w http.ResponseWriter, token string) {
	h := w.Header()
	existing := h.Values("Vary")
	for _, v := range existing {
		// "Vary" может быть списком
		for _, part := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return
			}
		}
	}
	h.Add("Vary", token)
}
