package queue

import (
	"context"
	"expvar"
	"pet-study/internal/entity"
	"sync"
)

var (
	queueRejectionsTotal *expvar.Int
	once                 sync.Once
)

type Queue struct {
	ch      chan WorkItem
	mu      sync.RWMutex
	stopped bool
}

type WorkItem struct {
	JobID   int64
	Payload entity.CreateUserInput
}

// StopAccepting is a strict post-return admission barrier. Enqueue calls that
// entered before StopAccepting acquired the barrier may complete, but once
// StopAccepting returns no Enqueue can succeed. The producer channel is not
// closed, so workers can continue draining already accepted work.
func (q *Queue) StopAccepting() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.stopped = true
}

func (q *Queue) Enqueue(ctx context.Context, item WorkItem) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.stopped {
		return ErrQueueStopped
	}
	if err := ctx.Err(); err != nil {
		return err
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
