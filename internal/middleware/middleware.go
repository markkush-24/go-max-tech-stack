package middleware

import (
	"log"
	"net/http"
	"pet-study/internal/httputils"
	"pet-study/internal/requestid"
	"pet-study/internal/security"
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

		sr, ok := w.(*statusRecorder)
		if !ok {
			sr = newStatusRecorder(w)
			w = sr
		}

		next.ServeHTTP(w, r)

		rid, _ := requestid.RequestID(r.Context())

		clientIP := "-"
		scheme := "-"
		if ri, ok := security.RequestInfoFromContext(r.Context()); ok {
			if ri.ClientIP != "" {
				clientIP = ri.ClientIP
			}
			if ri.Scheme != "" {
				scheme = ri.Scheme
			}
		}

		log.Printf(
			"[LOG] Method -- {%s},"+
				" UrlPath -- {%s},"+
				" Pattern -- {%s},"+
				" Status -- {%d},"+
				" Bytes -- {%d} ,"+
				" TimeSince -- {%v},"+
				" RequestID=%s"+
				" ClientIP=%s"+
				" Scheme=%s",
			r.Method, r.URL.Path, r.Pattern, sr.Status(), sr.Bytes(), time.Since(start), rid, clientIP, scheme,
		)
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
