package service

import (
	"context"
	"errors"
	"pet-study/internal/apperr"
	"pet-study/internal/entity"
	"testing"
)

func TestJobServiceAllowsExpectedTransitions(t *testing.T) {
	tests := []struct {
		name   string
		status entity.JobStatus
		call   func(context.Context, *JobService) error
		want   entity.JobStatus
	}{
		{
			name:   "queued to running",
			status: entity.JobQueued,
			call: func(ctx context.Context, svc *JobService) error {
				return svc.SetRunning(ctx, 1)
			},
			want: entity.JobRunning,
		},
		{
			name:   "running to succeeded",
			status: entity.JobRunning,
			call: func(ctx context.Context, svc *JobService) error {
				return svc.SetSucceeded(ctx, 1, entity.JobResult{UserID: 42})
			},
			want: entity.JobSucceeded,
		},
		{
			name:   "queued to failed",
			status: entity.JobQueued,
			call: func(ctx context.Context, svc *JobService) error {
				return svc.SetFailed(ctx, 1, entity.JobProblem{Detail: "shutdown"})
			},
			want: entity.JobFailed,
		},
		{
			name:   "running to failed",
			status: entity.JobRunning,
			call: func(ctx context.Context, svc *JobService) error {
				return svc.SetFailed(ctx, 1, entity.JobProblem{Detail: "failed"})
			},
			want: entity.JobFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newTransitionRepo(tt.status)
			svc := NewJobService(repo)

			if err := tt.call(context.Background(), svc); err != nil {
				t.Fatalf("transition error=%v", err)
			}
			if repo.job.Status != tt.want {
				t.Fatalf("status=%q want=%q", repo.job.Status, tt.want)
			}
		})
	}
}

func TestJobServiceRejectsUnexpectedSourceState(t *testing.T) {
	tests := []struct {
		name   string
		status entity.JobStatus
		call   func(context.Context, *JobService) error
	}{
		{
			name:   "queued cannot succeed",
			status: entity.JobQueued,
			call: func(ctx context.Context, svc *JobService) error {
				return svc.SetSucceeded(ctx, 1, entity.JobResult{UserID: 42})
			},
		},
		{
			name:   "succeeded cannot fail",
			status: entity.JobSucceeded,
			call: func(ctx context.Context, svc *JobService) error {
				return svc.SetFailed(ctx, 1, entity.JobProblem{Detail: "too late"})
			},
		},
		{
			name:   "failed cannot restart",
			status: entity.JobFailed,
			call: func(ctx context.Context, svc *JobService) error {
				return svc.SetRunning(ctx, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newTransitionRepo(tt.status)
			svc := NewJobService(repo)

			err := tt.call(context.Background(), svc)
			if err == nil {
				t.Fatal("transition error=nil, want conflict")
			}
			if !errors.Is(err, apperr.ErrJobTransitionConflict) {
				t.Fatalf("error=%v must wrap ErrJobTransitionConflict", err)
			}
			if !errors.Is(err, apperr.ErrConflict) {
				t.Fatalf("error=%v must wrap ErrConflict", err)
			}
			var conflict *apperr.JobTransitionConflictError
			if !errors.As(err, &conflict) {
				t.Fatalf("error=%T must be JobTransitionConflictError", err)
			}
			if repo.mutations != 0 {
				t.Fatalf("unexpected repository mutations=%d", repo.mutations)
			}
			if repo.job.Status != tt.status {
				t.Fatalf("status=%q want unchanged %q", repo.job.Status, tt.status)
			}
		})
	}
}

type transitionRepo struct {
	job       entity.Job
	mutations int
}

func newTransitionRepo(status entity.JobStatus) *transitionRepo {
	return &transitionRepo{
		job: entity.Job{
			ID:     1,
			Status: status,
		},
	}
}

func (r *transitionRepo) GetAll(context.Context) ([]*entity.Job, error) {
	job := r.job
	return []*entity.Job{&job}, nil
}

func (r *transitionRepo) GetByID(_ context.Context, id int64) (*entity.Job, error) {
	if id != r.job.ID {
		return nil, entity.ErrJobNotFound
	}
	job := r.job
	return &job, nil
}

func (r *transitionRepo) Save(_ context.Context, job *entity.Job) error {
	r.job = *job
	r.mutations++
	return nil
}

func (r *transitionRepo) Delete(_ context.Context, id int64) error {
	if id != r.job.ID {
		return entity.ErrJobNotFound
	}
	r.mutations++
	return nil
}

func (r *transitionRepo) SetRunning(_ context.Context, id int64) error {
	if id != r.job.ID {
		return entity.ErrJobNotFound
	}
	r.job.Status = entity.JobRunning
	r.mutations++
	return nil
}

func (r *transitionRepo) SetSucceeded(_ context.Context, id int64, res entity.JobResult) error {
	if id != r.job.ID {
		return entity.ErrJobNotFound
	}
	r.job.Status = entity.JobSucceeded
	r.job.Result = &res
	r.mutations++
	return nil
}

func (r *transitionRepo) SetFailed(_ context.Context, id int64, p entity.JobProblem) error {
	if id != r.job.ID {
		return entity.ErrJobNotFound
	}
	r.job.Status = entity.JobFailed
	r.job.Error = &p
	r.mutations++
	return nil
}

func (r *transitionRepo) FailActive(context.Context, entity.JobProblem) (int, error) {
	if !r.job.Status.IsTerminal() {
		r.job.Status = entity.JobFailed
		r.mutations++
		return 1, nil
	}
	return 0, nil
}
