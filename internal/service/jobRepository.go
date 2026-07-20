package service

import (
	"context"
	"pet-study/internal/entity"
)

type JobRepository interface {
	GetAll(ctx context.Context) ([]*entity.Job, error)
	GetByID(ctx context.Context, id int64) (*entity.Job, error)
	Save(ctx context.Context, job *entity.Job) error
	Delete(ctx context.Context, id int64) error

	// Transition methods are expected to be compare-and-set operations:
	// SetRunning applies queued -> running, SetSucceeded applies running -> succeeded,
	// and SetFailed applies queued/running -> failed. If the stored source status
	// does not match, repositories should return a typed transition-conflict error
	// without mutating the Job. TASK-010 tightens concrete storage implementations.
	SetRunning(ctx context.Context, id int64) error
	SetSucceeded(ctx context.Context, id int64, res entity.JobResult) error
	SetFailed(ctx context.Context, id int64, p entity.JobProblem) error

	// FailActive may atomically mark queued and running jobs as failed during
	// shutdown. Terminal jobs must remain immutable.
	FailActive(ctx context.Context, p entity.JobProblem) (int, error)
}
