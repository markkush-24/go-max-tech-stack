package routes

import (
	"errors"
	"fmt"
	"net/http"
	"pet-study/internal/entity"
	"pet-study/internal/httputils"
	"pet-study/internal/service"
	"strconv"
	"strings"
)

const prefixItems = "/api/v1/users/"
const prefixCollections = "/api/v1/users"

type UsersHandler struct {
	service *service.UserService
}

func NewUserHandler(service *service.UserService) *UsersHandler {
	return &UsersHandler{service: service}
}

func (h *UsersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, prefixItems) {
		idStr := strings.TrimPrefix(r.URL.Path, prefixItems)
		if idStr == "" || strings.Contains(idStr, "/") {
			_ = httputils.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}

		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			_ = httputils.WriteError(w, http.StatusBadRequest, "invalid_id", "id must be a positive integer")
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.getByID(w, r, id)
		default:
			methodNotAllowed(w, "GET")
			return
		}

	} else {
		if r.URL.Path != prefixCollections {
			_ = httputils.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.list(w, r)
		case http.MethodPost:
			h.create(w, r)
		default:
			methodNotAllowed(w, "GET, POST")
			return
		}
	}
}

func (h *UsersHandler) create(w http.ResponseWriter, r *http.Request) {
	httputils.LimitBody(w, r, 64<<10)
	var in entity.CreateUserInput

	if err := httputils.RequireJSONContentType(r); err != nil {
		_ = httputils.WriteError(w, http.StatusUnsupportedMediaType,
			"unsupported_media_type", "Content-Type must be application/json")
		return
	}

	if err := httputils.ParseJSON(r.Body, &in); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			_ = httputils.WriteError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body too large")
			return
		}

		// JSON decode errors -> 400 with stable messages
		var jre *httputils.JSONRequestError
		_ = errors.As(err, &jre) // если не распарсится — jre останется nil

		switch {
		case errors.Is(err, httputils.ErrJSONEmptyBody):
			_ = httputils.WriteError(w, http.StatusBadRequest,
				"empty_body", "request body must not be empty")

		case errors.Is(err, httputils.ErrJSONBadSyntax):
			_ = httputils.WriteError(w, http.StatusBadRequest,
				"invalid_json", "malformed JSON")

		case errors.Is(err, httputils.ErrJSONUnknownField):
			msg := "unknown field"
			if jre != nil && jre.Field != "" {
				msg = fmt.Sprintf("unknown field %q", jre.Field)
			}
			_ = httputils.WriteError(w, http.StatusBadRequest,
				"unknown_field", msg)

		case errors.Is(err, httputils.ErrJSONTypeMismatch):
			msg := "invalid JSON type"
			if jre != nil && jre.Field != "" {
				msg = fmt.Sprintf("invalid JSON type for field %q", jre.Field)
			}
			_ = httputils.WriteError(w, http.StatusBadRequest,
				"type_mismatch", msg)

		case errors.Is(err, httputils.ErrJSONTrailingData):
			_ = httputils.WriteError(w, http.StatusBadRequest,
				"trailing_data", "request body must contain a single JSON value")

		default:
			_ = httputils.WriteError(w, http.StatusBadRequest,
				"bad_request", "invalid request body")
		}
		return
	}
	if details := httputils.ValidateCreateUserInput(in); len(details) > 0 {
		_ = httputils.WriteError(w, http.StatusUnprocessableEntity,
			"validation_error", "validation failed", details...)
		return
	}

	u, err := h.service.CreateUser(r.Context(), &in)
	if err != nil {
		if errors.Is(err, service.ErrConflict) {
			_ = httputils.WriteError(w, http.StatusConflict, "conflict", "email already exists")
			return
		}
		_ = httputils.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/api/v1/users/%d", u.ID))
	httputils.WriteJSON(w, http.StatusCreated, u)
}

func (h *UsersHandler) getByID(w http.ResponseWriter, r *http.Request, id int) {
	u, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			_ = httputils.WriteError(w, http.StatusNotFound, "not_found", "user not found")
			return
		}

		_ = httputils.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	_ = httputils.WriteJSON(w, http.StatusOK, entity.UserDTO{
		ID: u.ID, Name: u.Name, Email: u.Email,
	})
}

func (h *UsersHandler) list(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.GetAllUsers(r.Context())
	if err != nil {
		_ = httputils.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return

	}

	usersDtos := make([]entity.UserDTO, 0, len(users))
	for _, u := range users {
		user := entity.UserDTO{
			ID: u.ID, Name: u.Name, Email: u.Email,
		}
		usersDtos = append(usersDtos, user)
	}

	_ = httputils.WriteJSON(w, http.StatusOK, usersDtos)
}

func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	_ = httputils.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
}
