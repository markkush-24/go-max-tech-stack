package httputils

import (
	"errors"
	"net/http"
	"pet-study/internal/service"
)

type APIError struct {
	Status  int
	Code    string
	Message string
}

var (
	errNotFound = APIError{Status: http.StatusNotFound, Code: "not_found", Message: "user not found"}
	errConflict = APIError{Status: http.StatusConflict, Code: "conflict", Message: "conflict"}
	errInternal = APIError{Status: http.StatusInternalServerError, Code: "internal", Message: "internal server error"}
)

func apiErrorFor(err error) APIError {
	switch {
	case errors.Is(err, service.ErrNotFound):
		return errNotFound
	case errors.Is(err, service.ErrConflict):
		return errConflict
	default:
		return errInternal
	}
}

func WriteServiceError(w http.ResponseWriter, err error) error {
	e := apiErrorFor(err)
	return WriteError(w, e.Status, e.Code, e.Message)
}
