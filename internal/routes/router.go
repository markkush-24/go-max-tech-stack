package routes

import (
	"fmt"
	"net/http"
	"pet-study/internal/entity"
	"pet-study/internal/httputils"
	"pet-study/internal/service"
	"strconv"
	"strings"
)

const (
	prefixItems       = "/api/v1/users/"
	prefixCollections = "/api/v1/users"
)

type UsersHandler struct {
	service *service.UserService
}

func NewUserHandler(service *service.UserService) *UsersHandler {
	return &UsersHandler{service: service}
}

func (h *UsersHandler) Handle(w http.ResponseWriter, r *http.Request) error {
	if strings.HasPrefix(r.URL.Path, prefixItems) {
		idStr := strings.TrimPrefix(r.URL.Path, prefixItems)
		if idStr == "" || strings.Contains(idStr, "/") {
			return service.ErrNotFound
		}

		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			return &httputils.BadRequestError{Detail: "id must be a positive integer"}
		}

		switch r.Method {
		case http.MethodGet:
			return h.getByID(w, r, id)
		default:
			return &httputils.MethodNotAllowedError{Allow: "GET"}
		}
	}

	if r.URL.Path != prefixCollections {
		return service.ErrNotFound
	}

	switch r.Method {
	case http.MethodGet:
		return h.list(w, r)
	case http.MethodPost:
		return h.create(w, r)
	default:
		return &httputils.MethodNotAllowedError{Allow: "GET, POST"}
	}
}

func (h *UsersHandler) create(w http.ResponseWriter, r *http.Request) error {
	httputils.LimitBody(w, r, 64<<10)

	if err := httputils.RequireJSONContentType(r); err != nil {
		return err
	}

	var in entity.CreateUserInput
	if err := httputils.ParseJSON(r.Body, &in); err != nil {
		return err
	}

	if details := httputils.ValidateCreateUserInput(in); len(details) > 0 {
		return &httputils.ValidationError{
			InvalidParams: httputils.ToInvalidParams(details),
		}
	}

	u, err := h.service.CreateUser(r.Context(), &in)
	if err != nil {
		return err
	}

	w.Header().Set("Location", fmt.Sprintf("/api/v1/users/%d", u.ID))
	return httputils.WriteJSON(w, http.StatusCreated, u)
}

func (h *UsersHandler) getByID(w http.ResponseWriter, r *http.Request, id int) error {
	u, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		return err
	}

	return httputils.WriteJSON(w, http.StatusOK, entity.UserDTO{
		ID:    u.ID,
		Name:  u.Name,
		Email: u.Email,
	})
}

func (h *UsersHandler) list(w http.ResponseWriter, r *http.Request) error {
	users, err := h.service.GetAllUsers(r.Context())
	if err != nil {
		return err
	}

	usersDtos := make([]entity.UserDTO, 0, len(users))
	for _, u := range users {
		usersDtos = append(usersDtos, entity.UserDTO{
			ID:    u.ID,
			Name:  u.Name,
			Email: u.Email,
		})
	}

	return httputils.WriteJSON(w, http.StatusOK, usersDtos)
}
