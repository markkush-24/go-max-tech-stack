package outbound

import (
	"context"
	"errors"
	"log"
	"net/url"
	"time"

	"pet-study/internal/metrics"
	"pet-study/internal/outbound/profile"
)

type InstrumentedProfileClient struct {
	next profile.Client
	m    *metrics.OutboundMetrics
	host string
	l    *log.Logger
}

func NewInstrumentedProfileClient(baseURL *url.URL, next profile.Client, l *log.Logger) *InstrumentedProfileClient {
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
			c.l.Printf("outbound request_id=%s host=%s route=%s user_id=%d status=%d err=%v latency=%s",
				requestID, c.host, route, userID, status, err, d)
		} else {
			c.l.Printf("outbound request_id=%s host=%s route=%s user_id=%d status=%d latency=%s",
				requestID, c.host, route, userID, status, d)
		}
	}

	return p, err
}
