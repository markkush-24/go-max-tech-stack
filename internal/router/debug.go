package router

import (
	"expvar"
	"net/http"
	"net/http/pprof"
)

func NewDebugRouter() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /debug/vars", expvar.Handler())

	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)

	return mux
}
