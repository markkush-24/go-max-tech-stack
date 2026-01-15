package health

import (
	"context"
	"sync/atomic"
)

type CheckFunc func(ctx context.Context) error

type Check struct {
	Name string
	Fn   CheckFunc
}

type Readiness struct {
	ready  atomic.Bool
	checks []Check
}

func NewReadiness(checks ...Check) *Readiness {
	cp := append([]Check(nil), checks...)
	return &Readiness{checks: cp}
}

func (r *Readiness) SetReady() {
	r.ready.Store(true)
}

func (r *Readiness) SetNotReady() {
	r.ready.Store(false)
}

func (r *Readiness) IsReady() bool {
	return r.ready.Load()
}
