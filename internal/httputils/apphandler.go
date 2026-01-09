package httputils

import (
	"log"
	"net/http"
)

type AppHandler func(w http.ResponseWriter, r *http.Request) error

func (h AppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		mp := MapError(r, err)

		for k, vv := range mp.Headers {
			w.Header().Del(k)
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}

		if mp.Log {
			log.Printf("request failed: method=%s path=%s err=%v", r.Method, r.URL.Path, err)
		}

		_ = WriteProblem(w, r, mp.Problem)
	}
}
