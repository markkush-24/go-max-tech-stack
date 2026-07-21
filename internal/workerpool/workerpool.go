package workerpool

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"pet-study/internal/apperr"
	"pet-study/internal/entity"
	"pet-study/internal/httputils"
	"pet-study/internal/metrics"
	"pet-study/internal/queue"
	"pet-study/internal/service"
	"pet-study/internal/stream"
	"sync"
	"sync/atomic"
	"time"
)

var ErrPoolNotRunning = errors.New("worker pool not running")
var ErrPoolStopping = errors.New("worker pool stopping")

const defaultStopRepairTimeout = 2 * time.Second

type StopOutcome struct {
	WorkersStopped     bool
	RepairAttempted    bool
	RepairedActiveJobs int
	RepairErr          error
}

func (o StopOutcome) Err() error {
	return o.RepairErr
}

// WorkerPool — lifecycle-компонент приложения для async jobs.
// Он хранит внутренний context только для времени жизни воркеров:
// этот context создаётся в Start из родительского app-context и не
// используется как request-scoped context.
type WorkerPool struct {
	queue        *queue.Queue
	jobService   *service.JobService
	userService  *service.UserService
	jobsObserver metrics.JobsObserver
	eventHub     *stream.Hub

	mu         sync.RWMutex
	generation *workerGeneration
	stopping   bool

	repairTimeout time.Duration
	lastOutcome   StopOutcome
	hasOutcome    bool
}

type workerGeneration struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	remaining atomic.Int64
	doneOnce  sync.Once
	stopOnce  sync.Once
	outcome   StopOutcome
}

func NewWorkerPool(
	q *queue.Queue,
	jobSvc *service.JobService,
	userSvc *service.UserService,
	metrics metrics.JobsObserver,
	eventHub *stream.Hub,
) *WorkerPool {
	return &WorkerPool{
		queue:         q,
		jobService:    jobSvc,
		userService:   userSvc,
		jobsObserver:  metrics,
		eventHub:      eventHub,
		repairTimeout: defaultStopRepairTimeout,
	}
}

func (wp *WorkerPool) CheckRunning(ctx context.Context) error {
	if !wp.IsRunning() {
		return ErrPoolNotRunning
	}
	return nil
}

func (wp *WorkerPool) Start(ctx context.Context, workers int) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	wp.settleStoppedLocked()

	if wp.generation != nil {
		if wp.stopping {
			return ErrPoolStopping
		}
		return nil
	}

	genCtx, cancel := context.WithCancel(ctx)
	gen := &workerGeneration{
		ctx:    genCtx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
	gen.remaining.Store(int64(workers))
	wp.generation = gen
	wp.stopping = false
	wp.lastOutcome = StopOutcome{}
	wp.hasOutcome = false

	for i := 0; i < workers; i++ {
		go wp.workerLoop(gen.ctx, gen)
	}

	if workers <= 0 {
		gen.closeDone()
	}

	return nil
}

func (wp *WorkerPool) Stop(ctx context.Context) error {
	wp.mu.Lock()
	gen := wp.generation
	if gen == nil {
		outcome := wp.lastOutcome
		hasOutcome := wp.hasOutcome
		wp.mu.Unlock()
		if hasOutcome {
			return outcome.Err()
		}
		return nil
	}
	firstStop := !wp.stopping
	wp.stopping = true
	wp.mu.Unlock()

	if firstStop {
		gen.cancel()
	}

	select {
	case <-gen.done:
		return wp.completeStop(ctx, gen)
	default:
	}

	select {
	case <-gen.done:
		return wp.completeStop(ctx, gen)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (wp *WorkerPool) IsRunning() bool {
	wp.mu.RLock()
	gen := wp.generation
	stopping := wp.stopping
	wp.mu.RUnlock()

	if gen == nil || stopping {
		return false
	}

	select {
	case <-gen.done:
		return false
	default:
		return true
	}
}

func (gen *workerGeneration) workerDone() {
	if gen.remaining.Add(-1) == 0 {
		gen.closeDone()
	}
}

func (gen *workerGeneration) closeDone() {
	gen.doneOnce.Do(func() {
		close(gen.done)
	})
}

func (wp *WorkerPool) settleStoppedLocked() {
	if wp.generation == nil {
		return
	}

	select {
	case <-wp.generation.done:
		if wp.stopping {
			return
		}
		wp.generation = nil
		wp.stopping = false
	default:
	}
}

func (wp *WorkerPool) completeStop(ctx context.Context, gen *workerGeneration) error {
	gen.stopOnce.Do(func() {
		repairCtx, cancel := wp.newRepairContext(ctx)
		defer cancel()
		gen.outcome = wp.repairActiveOnShutdown(repairCtx)
	})

	wp.mu.Lock()
	if wp.generation == gen {
		wp.generation = nil
		wp.stopping = false
		wp.lastOutcome = gen.outcome
		wp.hasOutcome = true
	}
	wp.mu.Unlock()

	return gen.outcome.Err()
}

func (wp *WorkerPool) LastStopOutcome() (StopOutcome, bool) {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	return wp.lastOutcome, wp.hasOutcome
}

func (wp *WorkerPool) newRepairContext(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := wp.repairTimeout
	if timeout <= 0 {
		timeout = defaultStopRepairTimeout
	}

	deadline := time.Now().Add(timeout)
	if parentDeadline, ok := ctx.Deadline(); ok && parentDeadline.Before(deadline) {
		deadline = parentDeadline
	}
	return context.WithDeadline(context.WithoutCancel(ctx), deadline)
}

func (wp *WorkerPool) workerLoop(ctx context.Context, gen *workerGeneration) {
	defer gen.workerDone()
	logger := slog.Default().With("component", "worker_pool")

	ch := wp.queue.Chan()

	for {
		select {
		case <-ctx.Done():
			return
		case item, ok := <-ch:
			if !ok {
				return
			}
			if ctx.Err() != nil {
				if err := wp.markJobFailed(ctx, item.JobID, service.ShutdownJobProblem()); err != nil {
					logger.Error("failed to mark job as failed during shutdown", "job_id", item.JobID, "err", err)
				}
				return
			}
			err := wp.jobService.SetRunning(ctx, item.JobID)
			if err != nil {
				if errors.Is(err, entity.ErrJobNotFound) {
					logger.Warn("job not found during transition to running", "job_id", item.JobID)
					continue
				} else {
					logger.Error("failed to set job running", "job_id", item.JobID, "err", err)
					continue
				}
			}
			wp.jobsObserver.IncRunning()
			wp.eventHub.Publish(item.JobID, stream.Event{
				Type:  string(entity.JobRunning),
				JobID: item.JobID,
				At:    time.Now(),
			})
			start := time.Now()

			user, userErr := wp.userService.CreateUser(ctx, &item.Payload)
			if userErr != nil {
				if err := wp.markJobFailed(ctx, item.JobID, ToJobProblem(userErr)); err != nil {
					logger.Error("failed to mark job as failed", "job_id", item.JobID, "err", err)
					continue
				}
				wp.jobsObserver.IncFailed()
				wp.jobsObserver.ObserveProcessing(time.Since(start))
				continue
			}

			res := entity.JobResult{UserID: int64(user.ID)}
			if err := wp.jobService.SetSucceeded(ctx, item.JobID, res); err != nil {
				logger.Error("failed to mark job as succeeded", "job_id", item.JobID, "err", err)
				continue
			}
			wp.jobsObserver.IncSucceeded()
			wp.eventHub.Publish(item.JobID, stream.Event{
				Type:  string(entity.JobSucceeded),
				JobID: item.JobID,
				At:    time.Now(),
				Data:  res,
			})
			wp.jobsObserver.ObserveProcessing(time.Since(start))
		}
	}
}

func (wp *WorkerPool) repairActiveOnShutdown(ctx context.Context) StopOutcome {
	outcome := StopOutcome{
		WorkersStopped:  true,
		RepairAttempted: true,
	}
	count, err := wp.jobService.FailActiveOnShutdown(ctx)
	if err != nil {
		outcome.RepairErr = err
		return outcome
	}
	outcome.RepairedActiveJobs = count
	if count > 0 {
		slog.Default().With("component", "worker_pool").Warn(
			"marked active jobs failed on shutdown",
			"count", count,
		)
	}
	return outcome
}

func (wp *WorkerPool) markJobFailed(ctx context.Context, id int64, problem entity.JobProblem) error {
	repairCtx, cancel := wp.newRepairContext(ctx)
	defer cancel()

	if err := wp.jobService.SetFailed(repairCtx, id, problem); err != nil {
		return err
	}
	// После отмены worker lifecycle-context нам всё ещё важно записать
	// terminal state job, поэтому здесь используется независимый bounded context.

	wp.eventHub.Publish(id, stream.Event{
		Type:  string(entity.JobFailed),
		JobID: id,
		At:    time.Now(),
		Data:  problem,
	})
	return nil
}

func ToJobProblem(err error) entity.JobProblem {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return service.ShutdownJobProblem()
	}

	var ve *httputils.ValidationError
	if errors.As(err, &ve) {
		jobInvalidParams := toJobInvalidParams(ve.InvalidParams)
		return entity.JobProblem{
			Title:         "validation failed",
			Detail:        "validation failed",
			Status:        http.StatusUnprocessableEntity,
			InvalidParams: jobInvalidParams,
		}
	}

	switch {
	case errors.Is(err, apperr.ErrConflict):
		return entity.JobProblem{
			Title:  "conflict",
			Detail: "conflict",
			Status: http.StatusConflict,
		}

	case errors.Is(err, apperr.ErrForbidden):
		return entity.JobProblem{
			Title:  "forbidden",
			Detail: "forbidden",
			Status: http.StatusForbidden,
		}
	}

	return entity.JobProblem{
		Title:  "internal server error",
		Detail: "internal server error",
		Status: http.StatusInternalServerError,
	}
}

func toJobInvalidParams(invalidParams []httputils.InvalidParam) []entity.JobInvalidParam {
	jobParams := make([]entity.JobInvalidParam, 0, len(invalidParams))
	for _, v := range invalidParams {
		jobInvalidParam := entity.JobInvalidParam{Name: v.Name, Reason: v.Reason}
		jobParams = append(jobParams, jobInvalidParam)
	}
	return jobParams
}
