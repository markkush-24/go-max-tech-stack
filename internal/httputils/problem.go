package httputils

import (
	"encoding/json"
	"net/http"
	"pet-study/internal/requestid"
)

type Problem struct {
	Type          string         `json:"type,omitempty"`
	Title         string         `json:"title,omitempty"`
	Status        int            `json:"status"`
	Detail        string         `json:"detail,omitempty"`
	Instance      string         `json:"instance,omitempty"`
	RequestID     string         `json:"request_id,omitempty"`
	InvalidParams []InvalidParam `json:"invalid_params,omitempty"`
}

type InvalidParam struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

func WriteProblem(w http.ResponseWriter, r *http.Request, p Problem) error {
	if p.Type == "" {
		p.Type = "about:blank"
	}
	if p.Status == 0 {
		p.Status = http.StatusInternalServerError
	}
	if p.Title == "" {
		p.Title = http.StatusText(p.Status)
	}
	if p.Instance == "" && r != nil {
		p.Instance = r.URL.Path
	}

	if p.RequestID == "" && r != nil {
		if rid, ok := requestid.RequestID(r.Context()); ok {
			p.RequestID = rid
		}
	}

	// гарантируем request-id в response header даже для ошибок
	if p.RequestID != "" && w.Header().Get(requestid.HeaderName) == "" {
		w.Header().Set(requestid.HeaderName, p.RequestID)
	}

	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(p.Status)
	return json.NewEncoder(w).Encode(p)
}

func ToInvalidParams(details []ErrorDetail) []InvalidParam {
	if len(details) == 0 {
		return nil
	}
	out := make([]InvalidParam, 0, len(details))
	for _, d := range details {
		out = append(out, InvalidParam{Name: d.Field, Reason: d.Rule})
	}
	return out
}
