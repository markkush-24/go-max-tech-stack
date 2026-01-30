package workerpool

import (
	"context"
	"errors"
	"log"
	"net/http"
	"pet-study/internal/entity"
	"pet-study/internal/httputils"
	"pet-study/internal/queue"
	"pet-study/internal/service"
	"sync"
)

var ErrPoolNotRunning = errors.New("worker pool not running")

type WorkerPool struct {
	queue       *queue.Queue
	jobService  *service.JobService
	userService *service.UserService

	mu      sync.Mutex
	running bool

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewWorkerPool(q *queue.Queue, jobSvc *service.JobService, userSvc *service.UserService) *WorkerPool {
	return &WorkerPool{
		queue:       q,
		jobService:  jobSvc,
		userService: userSvc,
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
			err := wp.jobService.SetRunning(wp.ctx, item.JobID)
			if err != nil {
				if errors.Is(err, entity.ErrJobNotFound) {
					log.Printf("job not found %d", item.JobID)
					continue
				} else {
					log.Printf("Error when attempt SetRunning status for job  %d", item.JobID)
					continue
				}
			}
			user, userErr := wp.userService.CreateUser(wp.ctx, &item.Payload)
			if userErr != nil {
				err := wp.jobService.SetFailed(wp.ctx, item.JobID, ToJobProblem(userErr))
				if err != nil {
					log.Printf("Error when attempt setFailed status for job %d , error = %v", item.JobID, err)
				}
				log.Printf("Error when attempt to create user jobID = %v , err = %v", item.JobID, userErr)
				continue
			} else {
				succeedErr := wp.jobService.SetSucceeded(wp.ctx, item.JobID, entity.JobResult{UserID: int64(user.ID)})
				if succeedErr != nil {
					log.Printf("Error when attempt SetSucceeded status for job  %d, error = %v", item.JobID, succeedErr)
				}
			}
		}
	}
}

func ToJobProblem(err error) entity.JobProblem {
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
	case errors.Is(err, service.ErrConflict):
		return entity.JobProblem{
			Title:  "conflict",
			Detail: "conflict",
			Status: http.StatusConflict,
		}

	case errors.Is(err, service.ErrForbidden):
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
