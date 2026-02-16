package httpapi

import (
	"net/http"
)

type UsersProfileAPI interface {
	GetUserProfile(w http.ResponseWriter, r *http.Request, userID int64) error
}
