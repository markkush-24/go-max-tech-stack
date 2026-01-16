package middleware

import (
	"log"
	"net/http"
	"pet-study/internal/httputils"
	"pet-study/internal/requestid"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusRecorder) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rid, _ := requestid.RequestID(r.Context())
		sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			log.Printf(
				"[LOG] Method -- {%s},"+
					" UrlPath -- {%s},"+
					" RequestID -- {%s},"+
					" Status -- {%d},"+
					" Bytes -- {%d} ,"+
					" TimeSince -- {%v}",
				r.Method, r.URL.Path, rid, sr.status, sr.bytes, time.Since(start))
		}()
		next.ServeHTTP(sr, r)
	})
}

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				rid, _ := requestid.RequestID(r.Context())
				log.Printf("panic request_id=%s err=%v", rid, err)
				_ = httputils.WriteProblem(w, r, httputils.Problem{
					Status:    http.StatusInternalServerError,
					Detail:    "internal server error",
					RequestID: rid,
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
