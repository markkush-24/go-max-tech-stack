package httpapi

import "net/http"

type UsersAPI interface {
	List(w http.ResponseWriter, r *http.Request) error
	Create(w http.ResponseWriter, r *http.Request) error
	GetByID(w http.ResponseWriter, r *http.Request, id int) error
}
