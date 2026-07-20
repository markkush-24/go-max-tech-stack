package workerpool

import (
	"context"
	"errors"
	"pet-study/internal/entity"
	"pet-study/internal/queue"
	"pet-study/internal/service"
	"pet-study/internal/store/jobrepo"
	"pet-study/internal/store/userrepo"
	"pet-study/internal/stream"
	"sync"
	"testing"
	"time"
)

func TestStopTimeoutPreventsRestartUntilGenerationDone(t *testing.T) {
	userRepo := newBlockingUserRepo()
	pool, q, jobSvc := newTestPool(userRepo)

	if err := pool.Start(context.Background(), 1); err != nil {
		t.Fatalf("Start: %v", err)
	}

	job := entity.Job{Status: entity.JobQueued, OwnerUserID: 1}
	if err := jobSvc.Save(context.Background(), &job); err != nil {
		t.Fatalf("Save job: %v", err)
	}
	if err := q.Enqueue(context.Background(), queue.WorkItem{
		JobID: job.ID,
		Payload: entity.CreateUserInput{
			Name:  "blocked",
			Email: "blocked@example.com",
			Age:   21,
		},
	}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	waitClosed(t, userRepo.saveStarted, "worker to enter blocking Save")

	stopCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	err := pool.Stop(stopCtx)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Stop error=%v, want DeadlineExceeded", err)
	}

	if err := pool.Start(context.Background(), 1); !errors.Is(err, ErrPoolStopping) {
		t.Fatalf("Start while stopping error=%v, want ErrPoolStopping", err)
	}

	close(userRepo.release)
	if err := stopPoolWithTimeout(pool, time.Second); err != nil {
		t.Fatalf("Stop after worker release: %v", err)
	}

	if err := pool.Start(context.Background(), 1); err != nil {
		t.Fatalf("Start after prior generation stopped: %v", err)
	}
	if err := stopPoolWithTimeout(pool, time.Second); err != nil {
		t.Fatalf("final Stop: %v", err)
	}
}

func TestStartCreatesFreshGenerationAfterStop(t *testing.T) {
	pool, _, _ := newTestPool(userrepo.NewMemoryUserRepository())

	if err := pool.Start(context.Background(), 1); err != nil {
		t.Fatalf("Start: %v", err)
	}
	old := currentGeneration(t, pool)
	if old == nil {
		t.Fatal("old generation=nil")
	}

	if err := stopPoolWithTimeout(pool, time.Second); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if old.ctx.Err() == nil {
		t.Fatal("old generation context was not canceled")
	}

	if err := pool.Start(context.Background(), 1); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	next := currentGeneration(t, pool)
	if next == nil {
		t.Fatal("new generation=nil")
	}
	if next == old {
		t.Fatal("Start reused the previous generation")
	}
	if next.ctx.Err() != nil {
		t.Fatalf("new generation context err=%v, want nil", next.ctx.Err())
	}

	if err := stopPoolWithTimeout(pool, time.Second); err != nil {
		t.Fatalf("final Stop: %v", err)
	}
}

func TestConcurrentStopCallsShareOneStoppingGeneration(t *testing.T) {
	userRepo := newBlockingUserRepo()
	pool, q, jobSvc := newTestPool(userRepo)

	if err := pool.Start(context.Background(), 1); err != nil {
		t.Fatalf("Start: %v", err)
	}

	job := entity.Job{Status: entity.JobQueued, OwnerUserID: 1}
	if err := jobSvc.Save(context.Background(), &job); err != nil {
		t.Fatalf("Save job: %v", err)
	}
	if err := q.Enqueue(context.Background(), queue.WorkItem{
		JobID: job.ID,
		Payload: entity.CreateUserInput{
			Name:  "blocked",
			Email: "concurrent-stop@example.com",
			Age:   21,
		},
	}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	waitClosed(t, userRepo.saveStarted, "worker to enter blocking Save")

	longStopErr := make(chan error, 1)
	go func() {
		longStopErr <- stopPoolWithTimeout(pool, time.Second)
	}()
	waitStopping(t, pool)

	shortCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	err := pool.Stop(shortCtx)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second Stop error=%v, want DeadlineExceeded", err)
	}
	if err := pool.Start(context.Background(), 1); !errors.Is(err, ErrPoolStopping) {
		t.Fatalf("Start while concurrent Stop is waiting error=%v, want ErrPoolStopping", err)
	}

	close(userRepo.release)
	if err := <-longStopErr; err != nil {
		t.Fatalf("long Stop: %v", err)
	}
	if currentGeneration(t, pool) != nil {
		t.Fatal("generation still present after successful Stop")
	}
}

func TestStartIsIdempotentWithinRunningGeneration(t *testing.T) {
	pool, _, _ := newTestPool(userrepo.NewMemoryUserRepository())

	if err := pool.Start(context.Background(), 1); err != nil {
		t.Fatalf("Start: %v", err)
	}
	first := currentGeneration(t, pool)

	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- pool.Start(context.Background(), 1)
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Start error=%v, want nil", err)
		}
	}
	if got := currentGeneration(t, pool); got != first {
		t.Fatal("concurrent Start replaced the running generation")
	}

	if err := stopPoolWithTimeout(pool, time.Second); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

type noopJobsObserver struct{}

func (noopJobsObserver) IncQueued()                      {}
func (noopJobsObserver) IncRunning()                     {}
func (noopJobsObserver) IncSucceeded()                   {}
func (noopJobsObserver) IncFailed()                      {}
func (noopJobsObserver) ObserveProcessing(time.Duration) {}

type blockingUserRepo struct {
	saveStarted chan struct{}
	release     chan struct{}
	startOnce   sync.Once
}

func newBlockingUserRepo() *blockingUserRepo {
	return &blockingUserRepo{
		saveStarted: make(chan struct{}),
		release:     make(chan struct{}),
	}
}

func (r *blockingUserRepo) GetAll(context.Context) ([]*entity.User, error) {
	return nil, nil
}

func (r *blockingUserRepo) GetByID(context.Context, int) (*entity.User, error) {
	return nil, entity.ErrUserNotFound
}

func (r *blockingUserRepo) Save(ctx context.Context, user *entity.User) error {
	r.startOnce.Do(func() {
		close(r.saveStarted)
	})

	<-r.release
	return ctx.Err()
}

func (r *blockingUserRepo) Delete(context.Context, int) error {
	return entity.ErrUserNotFound
}

func (r *blockingUserRepo) ExistsByEmail(context.Context, string) (bool, error) {
	return false, nil
}

func newTestPool(userRepo service.UserRepository) (*WorkerPool, *queue.Queue, *service.JobService) {
	jobSvc := service.NewJobService(jobrepo.NewMemoryJobRepository())
	userSvc := service.NewUserService(userRepo)
	q := queue.New(4)
	return NewWorkerPool(q, jobSvc, userSvc, noopJobsObserver{}, stream.NewHub(16)), q, jobSvc
}

func currentGeneration(t *testing.T, pool *WorkerPool) *workerGeneration {
	t.Helper()

	pool.mu.RLock()
	defer pool.mu.RUnlock()
	return pool.generation
}

func waitStopping(t *testing.T, pool *WorkerPool) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for {
		pool.mu.RLock()
		stopping := pool.stopping
		pool.mu.RUnlock()
		if stopping {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("timeout waiting for pool to enter stopping state")
		}
		time.Sleep(time.Millisecond)
	}
}

func waitClosed(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for %s", what)
	}
}

func stopPoolWithTimeout(pool *WorkerPool, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return pool.Stop(ctx)
}
