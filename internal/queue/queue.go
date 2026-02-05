package queue

import (
	"context"
	"expvar"
	"pet-study/internal/entity"
	"sync"
	"sync/atomic"
)

var (
	queueRejectionsTotal *expvar.Int
	once                 sync.Once
)

type Queue struct {
	ch        chan WorkItem
	closed    atomic.Bool
	closeOnce sync.Once
}

type WorkItem struct {
	JobID   int64
	Payload entity.CreateUserInput
}

func (q *Queue) StopAccepting() {
	q.closed.Store(true)
}

func (q *Queue) Enqueue(ctx context.Context, item WorkItem) error {
	if q.closed.Load() == true {
		return ErrQueueStopped
	}
	select {
	case q.ch <- item:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		if queueRejectionsTotal != nil {
			queueRejectionsTotal.Add(1)
		}
		return ErrQueueFull
	}
}

func New(capacity int) *Queue {
	q := &Queue{ch: make(chan WorkItem, capacity)}

	once.Do(func() {
		queueRejectionsTotal = expvar.NewInt("queue_rejections_total")

		expvar.Publish("queue_depth", expvar.Func(func() any {
			return len(q.ch)
		}))
	})

	return q
}

func (q *Queue) Chan() <-chan WorkItem {
	return q.ch
}
