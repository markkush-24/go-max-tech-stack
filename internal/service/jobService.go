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
