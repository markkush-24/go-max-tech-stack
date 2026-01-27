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
}
