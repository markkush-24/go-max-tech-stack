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

// Пробрасываем опциональные интерфейсы (на будущее; не мешает сейчас).
func (sr *statusRecorder) Flush() {
	if f, ok := sr.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
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
