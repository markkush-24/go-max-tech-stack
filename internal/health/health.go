package health

import "sync/atomic"

type Readiness struct {
	v atomic.Bool
}

func NewReadiness() *Readiness {
	return &Readiness{v: atomic.Bool{}}
}

func (r *Readiness) SetReady() {
	r.v.Store(true)
}

func (r *Readiness) SetNotReady() {
	r.v.Store(false)
}

func (r *Readiness) IsReady() bool {
	return r.v.Load()
}
