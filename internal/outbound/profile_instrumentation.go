package outbound

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"pet-study/internal/config"
	"pet-study/internal/metrics"
	"pet-study/internal/outbound/profile"
)

type InstrumentedProfileClient struct {
	next profile.Client
	m    *metrics.OutboundMetrics
	host string
	l    *slog.Logger
}

func NewInstrumentedProfileClient(baseURL *url.URL, next profile.Client, l *slog.Logger) *InstrumentedProfileClient {
	return &InstrumentedProfileClient{
		next: next,
		m:    metrics.DefaultOutbound(),
		host: baseURL.Host,
		l:    l,
	}
}

func (c *InstrumentedProfileClient) FetchProfile(ctx context.Context, userID int64, requestID string) (profile.Profile, error) {
	const (
		method       = http.MethodGet
		route        = "/profiles/{user_id}"
		metricsRoute = method + " " + route
	)

	start := time.Now()
	p, err := c.next.FetchProfile(ctx, userID, requestID)
	d := time.Since(start)

	status := 200
	if err != nil {
		status = 0
		var pe *profile.Error
		if errors.As(err, &pe) && pe.Status != 0 {
			status = pe.Status
		}
		c.m.IncError(profile.KindLabel(err))
	}
	c.m.Observe(c.host, metricsRoute, status, d)

	if c.l != nil {
		if err != nil {
			c.l.Warn(
				"outbound request failed",
				config.LogFieldRequestID, requestID,
				"host", c.host,
				config.LogFieldMethod, method,
				config.LogFieldRoute, route,
				"user_id", userID,
				config.LogFieldStatus, status,
				config.LogFieldDurationMS, d.Milliseconds(),
				config.LogFieldError, err,
			)
		} else {
			c.l.Info(
				"outbound request completed",
				config.LogFieldRequestID, requestID,
				"host", c.host,
				config.LogFieldMethod, method,
				config.LogFieldRoute, route,
				"user_id", userID,
				config.LogFieldStatus, status,
				config.LogFieldDurationMS, d.Milliseconds(),
			)
		}
	}

	return p, err
}
