package router

import "net/http"

func NewRoot(app http.Handler, health http.Handler) http.Handler {
	mux := http.NewServeMux()

	// Весь «бизнес»-трафик отдаём app-роутеру.
	// Он сам знает про /api/v1/... (у тебя уже так сделано).
	mux.Handle("/", app)

	// Health оставляем отдельными точками входа без middleware.
	mux.Handle("/livez", health)
	mux.Handle("/readyz", health)

	return mux
}
