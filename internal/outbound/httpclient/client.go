package httpclient

import (
	"net/http"
	"pet-study/internal/config"
)

func New(cfg config.OutboundConfig) (*http.Client, *http.Transport) {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.IdleConnTimeout = cfg.Transport.IdleConnTimeout
	tr.MaxIdleConns = cfg.Transport.MaxIdleConns
	tr.MaxIdleConnsPerHost = cfg.Transport.MaxIdleConnsPerHost
	tr.MaxConnsPerHost = cfg.Transport.MaxConnsPerHost
	tr.TLSHandshakeTimeout = cfg.Transport.TLSHandshakeTimeout
	tr.ResponseHeaderTimeout = cfg.Transport.ResponseHeaderTimeout

	c := &http.Client{Transport: tr}

	return c, tr
}
