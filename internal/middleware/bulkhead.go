package middleware

import (
	"expvar"
	"net/http"
	"pet-study/internal/httputils"
	"sync"
	"sync/atomic"
)

var (
	initOnce        sync.Once
	rejectionsTotal *expvar.Int
	inFlight        atomic.Int64
)

type BulkheadAPI struct {
	sem chan struct{}
}

func NewBulkhead(maxParallel int) *BulkheadAPI {
	initOnce.Do(func() {
		rejectionsTotal = expvar.NewInt("bulkhead_rejections_total")
		expvar.Publish("bulkhead_in_flight", expvar.Func(func() any {
			return inFlight.Load()
		}))
	})

	return &BulkheadAPI{
		sem: make(chan struct{}, maxParallel),
	}
}

func (bh *BulkheadAPI) Bulkhead(next httputils.AppHandler) httputils.AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		select {
		case bh.sem <- struct{}{}:
			inFlight.Add(1)
			defer inFlight.Add(-1)
			defer func() { <-bh.sem }()
			return next(w, r)
		default:
			rejectionsTotal.Add(1)
			return httputils.ErrBulkheadRejected
		}
	}
}
