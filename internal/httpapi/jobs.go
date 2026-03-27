package httpapi

import "net/http"

type JobsAPI interface {
	GetByID(w http.ResponseWriter, r *http.Request, id int) error
	Events(w http.ResponseWriter, r *http.Request, id int) error
}
