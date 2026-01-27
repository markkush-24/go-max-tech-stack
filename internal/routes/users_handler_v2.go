package routes

import (
	"fmt"
	"net/http"
	"pet-study/internal/entity"
	"pet-study/internal/httputils"
	"pet-study/internal/service"
)

type UsersV2Handler struct {
	userService *service.UserService
	jobService  *service.JobService
}

func NewUserV2Handler(userService *service.UserService, jobService *service.JobService) *UsersV2Handler {
	return &UsersV2Handler{userService: userService, jobService: jobService}
}

func (h *UsersV2Handler) Create(w http.ResponseWriter, r *http.Request) error {
	httputils.LimitBody(w, r, 64<<10)

	if err := httputils.RequireJSONContentType(r); err != nil {
		return err
	}

	var in entity.CreateUserInputV2
	if err := httputils.ParseJSON(r.Body, &in); err != nil {
		return err
	}

	if details := httputils.ValidateCreateUserInputV2(in); len(details) > 0 {
		return &httputils.ValidationError{InvalidParams: httputils.ToInvalidParams(details)}
	}

	v1in := entity.MapCreateV2ToV1(in)

	created, err := h.userService.CreateUser(r.Context(), &v1in)
	if err != nil {
		return err
	}

	w.Header().Set("Location", fmt.Sprintf("/api/v2/users/%d", created.ID))
	return httputils.WriteJSON(w, http.StatusCreated, entity.MapUserDTOToV2(created))
}

func (h *UsersV2Handler) GetByID(w http.ResponseWriter, r *http.Request, id int) error {
	u, err := h.userService.GetByID(r.Context(), id)
	if err != nil {
		return err
	}

	return httputils.WriteJSON(w, http.StatusOK, entity.UserDTO{
		ID:    u.ID,
		Name:  u.Name,
		Email: u.Email,
	})
}

func (h *UsersV2Handler) List(w http.ResponseWriter, r *http.Request) error {
	users, err := h.userService.GetAllUsers(r.Context())
	if err != nil {
		return err
	}
	return httputils.WriteJSON(w, http.StatusOK, entity.MapUsersToV2(users))
}
