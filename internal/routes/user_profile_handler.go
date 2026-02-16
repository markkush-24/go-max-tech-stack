package routes

import (
	"net/http"
	"pet-study/internal/entity"
	"pet-study/internal/httputils"
	"pet-study/internal/requestid"
	"pet-study/internal/service"
)

type UsersProfileHandler struct {
	userProfileService *service.UserProfileService
}

func NewUsersProfileHandler(userProfileService *service.UserProfileService) *UsersProfileHandler {
	return &UsersProfileHandler{userProfileService: userProfileService}
}

func (h *UsersProfileHandler) GetUserProfile(
	w http.ResponseWriter, r *http.Request, userId int64) error {
	rid, ok := requestid.RequestID(r.Context())
	if !ok {
		rid = r.Header.Get(requestid.HeaderName)
	}
	u, p, err := h.userProfileService.GetUserWithProfile(r.Context(), userId, rid)
	if err != nil {
		return err
	}
	return httputils.WriteNegotiated(w, r, http.StatusOK, entity.UserProfileDTO{
		ID:    int64(u.ID),
		Name:  u.Name,
		Email: u.Email,
		Profile: entity.ProfileDTO{
			Bio:  p.Bio,
			City: p.City,
		},
	}, nil)
}
