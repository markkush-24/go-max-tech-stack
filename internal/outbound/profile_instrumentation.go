package outbound

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"time"

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
	const route = "GET /profiles/{user_id}"

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
	c.m.Observe(c.host, route, status, d)

	if c.l != nil {
		if err != nil {
			c.l.Warn(
				"outbound request failed",
				"component", "outbound_profile",
				"request_id", requestID,
				"host", c.host,
				"route", route,
				"user_id", userID,
				"status", status,
				"latency_ms", d.Milliseconds(),
				"err", err,
			)
		} else {
			c.l.Info(
				"outbound request completed",
				"component", "outbound_profile",
				"request_id", requestID,
				"host", c.host,
				"route", route,
				"user_id", userID,
				"status", status,
				"latency_ms", d.Milliseconds(),
			)
		}
	}

	return p, err
}
