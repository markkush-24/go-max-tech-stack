package workerpool

import (
	"context"
	"errors"
	"pet-study/internal/queue"
	"pet-study/internal/service"
	"sync"
)

var ErrPoolNotRunning = errors.New("worker pool not running")

type WorkerPool struct {
	queue      *queue.Queue
	jobService *service.JobService

	mu      sync.Mutex
	running bool

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewWorkerPool(q *queue.Queue, jobSvc *service.JobService) *WorkerPool {
	return &WorkerPool{
		queue:      q,
		jobService: jobSvc,
	}
}

func (wp *WorkerPool) CheckRunning(ctx context.Context) error {
	if !wp.IsRunning() {
		return errors.New("worker pool not running")
	}
	return nil
}

func (wp *WorkerPool) Start(workers int) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.running {
		return nil
	}

	wp.ctx, wp.cancel = context.WithCancel(context.Background())
	wp.running = true

	for i := 0; i < workers; i++ {
		wp.wg.Add(1)
		go wp.workerLoop()
	}

	return nil
}

func (wp *WorkerPool) Stop(ctx context.Context) error {
	wp.mu.Lock()
	if !wp.running {
		wp.mu.Unlock()
		return ErrPoolNotRunning
	}
	cancel := wp.cancel
	wp.running = false
	wp.mu.Unlock()

	cancel()

	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (wp *WorkerPool) IsRunning() bool {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	return wp.running
}

func (wp *WorkerPool) workerLoop() {
	defer wp.wg.Done()

	ch := wp.queue.Chan()

	for {
		select {
		case <-wp.ctx.Done():
			return
		case item, ok := <-ch:
			if !ok {
				return
			}
			_ = wp.jobService.SetRunning(wp.ctx, item.JobID)
		}
	}
}
