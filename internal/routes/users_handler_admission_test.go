package routes

import (
	"context"
	"pet-study/internal/entity"
	"pet-study/internal/service"
	"testing"
)

func TestDeleteJobAfterFailedEnqueueUsesDetachedBoundedContext(t *testing.T) {
	repo := &rollbackJobRepo{}
	jobSvc := service.NewJobService(repo)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := deleteJobAfterFailedEnqueue(ctx, jobSvc, 42); err != nil {
		t.Fatalf("deleteJobAfterFailedEnqueue: %v", err)
	}

	if repo.deletedID != 42 {
		t.Fatalf("deletedID=%d want=42", repo.deletedID)
	}
	if repo.deleteCanceledAtStart {
		t.Fatal("rollback Delete received already-canceled request context")
	}
	if !repo.deleteHadDeadline {
		t.Fatal("rollback Delete context had no bounded deadline")
	}
}

type rollbackJobRepo struct {
	deletedID             int64
	deleteCanceledAtStart bool
	deleteHadDeadline     bool
}

func (r *rollbackJobRepo) GetAll(context.Context) ([]*entity.Job, error) {
	return nil, nil
}

func (r *rollbackJobRepo) GetByID(context.Context, int64) (*entity.Job, error) {
	return nil, entity.ErrJobNotFound
}

func (r *rollbackJobRepo) Save(context.Context, *entity.Job) error {
	return nil
}

func (r *rollbackJobRepo) Delete(ctx context.Context, id int64) error {
	r.deletedID = id
	r.deleteCanceledAtStart = ctx.Err() != nil
	_, r.deleteHadDeadline = ctx.Deadline()
	return nil
}

func (r *rollbackJobRepo) SetRunning(context.Context, int64) error {
	return nil
}

func (r *rollbackJobRepo) SetSucceeded(context.Context, int64, entity.JobResult) error {
	return nil
}

func (r *rollbackJobRepo) SetFailed(context.Context, int64, entity.JobProblem) error {
	return nil
}

func (r *rollbackJobRepo) FailActive(context.Context, entity.JobProblem) (int, error) {
	return 0, nil
}
