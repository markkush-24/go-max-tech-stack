package service

import (
	"context"
	"pet-study/internal/entity"
)

type JobService struct {
	repo JobRepository
}

func NewJobService(repo JobRepository) *JobService {
	return &JobService{repo: repo}
}

func (s *JobService) Save(ctx context.Context, job *entity.Job) error {
	return s.repo.Save(ctx, job)
}

func (s *JobService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *JobService) GetAll(ctx context.Context) ([]*entity.Job, error) {
	return s.repo.GetAll(ctx)
}
func (s *JobService) GetByID(ctx context.Context, id int64) (*entity.Job, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *JobService) SetRunning(ctx context.Context, id int64) error {
	return s.repo.SetRunning(ctx, id)
}
func (s *JobService) SetSucceeded(ctx context.Context, id int64, res entity.JobResult) error {
	return s.repo.SetSucceeded(ctx, id, res)
}
func (s *JobService) SetFailed(ctx context.Context, id int64, p entity.JobProblem) error {
	return s.repo.SetFailed(ctx, id, p)
}
