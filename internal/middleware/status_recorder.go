package middleware

import (
	"bufio"
	"errors"
	"net"
	"net/http"
)

func newStatusRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{ResponseWriter: w, status: http.StatusOK}
}

func (sr *statusRecorder) Status() int { return sr.status }
func (sr *statusRecorder) Bytes() int  { return sr.bytes }

func (sr *statusRecorder) Unwrap() http.ResponseWriter {
	return sr.ResponseWriter
}

// Пробрасываем опциональные интерфейсы для streaming/delegation.
func (sr *statusRecorder) Flush() {
	_ = sr.FlushError()
}

func (sr *statusRecorder) FlushError() error {
	if f, ok := sr.ResponseWriter.(interface{ FlushError() error }); ok {
		if !sr.wroteHeader {
			sr.WriteHeader(http.StatusOK)
		}
		return f.FlushError()
	}
	if f, ok := sr.ResponseWriter.(http.Flusher); ok {
		if !sr.wroteHeader {
			sr.WriteHeader(http.StatusOK)
		}
		f.Flush()
		return nil
	}
	return http.ErrNotSupported
}

func (sr *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := sr.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijacker not supported")
	}
	return h.Hijack()
}

func (sr *statusRecorder) Push(target string, opts *http.PushOptions) error {
	if p, ok := sr.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}
