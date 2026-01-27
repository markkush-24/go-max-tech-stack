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

	SetRunning(ctx context.Context, id int64) error
	SetSucceeded(ctx context.Context, id int64, res entity.JobResult) error
	SetFailed(ctx context.Context, id int64, p entity.JobProblem) error
}
