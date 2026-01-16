package router

import "net/http"

func NewRoot(app http.Handler, health http.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/", app)

	mux.Handle("/livez", health)
	mux.Handle("/readyz", health)

	return mux
}
