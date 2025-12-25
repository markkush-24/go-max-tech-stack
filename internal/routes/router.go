package routes

import (
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
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
	service   *service.UserService
	validator *validator.Validate
}

func NewUserHandler(service *service.UserService, validator *validator.Validate) *UsersHandler {
	return &UsersHandler{service: service, validator: validator}
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
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var in entity.CreateUserInput
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		_ = httputils.WriteError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return
	}
	if err := httputils.ParseJSON(r, &in); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			_ = httputils.WriteError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body too large")
			return
		}

		_ = httputils.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	if err := h.validator.Struct(in); err != nil {
		ves := err.(validator.ValidationErrors)
		var b strings.Builder
		for _, e := range ves {
			fmt.Fprintf(&b, "%s: %s; ", e.Field(), e.Tag())
		}
		httputils.WriteError(w, http.StatusUnprocessableEntity, "validation_error", strings.TrimSpace(b.String()))
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
		_ = httputils.WriteError(w, httputils.StatusFor(err), "not_found", "user not found")
		return
	}

	_ = httputils.WriteJSON(w, http.StatusOK, entity.UserDTO{
		ID: u.ID, Name: u.Name, Email: u.Email,
	})
}

func (h *UsersHandler) list(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
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
