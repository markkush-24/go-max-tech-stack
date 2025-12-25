package httputils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func ParseJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return fmt.Errorf("missing request body")
	}
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	if dec.More() {
		return fmt.Errorf("unexpected data after JSON value")
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
