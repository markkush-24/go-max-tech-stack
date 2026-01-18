package router

import "net/http"

func NewRoot(app http.Handler, health http.Handler, debug http.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/", app)

	mux.Handle("/livez", health)
	mux.Handle("/readyz", health)

	if debug != nil {
		mux.Handle("/debug/", debug)
	}

	return mux
}
