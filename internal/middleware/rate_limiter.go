package middleware

import (
	"context"
	"expvar"
	"golang.org/x/time/rate"
	"net/http"
	"pet-study/internal/httputils"
	"sync"
)

var (
	once  sync.Once
	total *expvar.Int
)

func (api *RateLimitedAPI) RateLimiter(next httputils.AppHandler) httputils.AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := api.Process(r.Context()); err != nil {
			return err
		}
		return next(w, r)
	}
}

type RateLimitedAPI struct {
	limiter *rate.Limiter
	total   *expvar.Int
}

func NewRateLimitedAPI(rps float64, burst int) *RateLimitedAPI {
	once.Do(func() {
		total = expvar.NewInt("rate_limited_total")
	})

	return &RateLimitedAPI{
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
		total:   total,
	}
}

func (api *RateLimitedAPI) Process(ctx context.Context) error {
	res := api.limiter.Reserve()

	if !res.OK() {
		api.total.Add(1)
		return &httputils.RateLimitError{RetryAfter: 0}
	}

	d := res.Delay()
	if d == 0 {
		return nil
	}

	if d > 0 {
		res.Cancel()
		api.total.Add(1)
		return &httputils.RateLimitError{RetryAfter: d}
	}
	return nil
}
