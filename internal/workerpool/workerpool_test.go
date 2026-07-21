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
	"runtime"
	"sync"
	"sync/atomic"
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

func TestRepeatedStopReturnsSameOutcomeAndRepairsOnce(t *testing.T) {
	jobRepo := newCountingJobRepo()
	pool, _, jobSvc := newTestPoolWithJobRepo(userrepo.NewMemoryUserRepository(), jobRepo)

	queued := entity.Job{Status: entity.JobQueued, OwnerUserID: 1}
	if err := jobSvc.Save(context.Background(), &queued); err != nil {
		t.Fatalf("Save queued job: %v", err)
	}
	succeeded := entity.Job{
		Status:      entity.JobSucceeded,
		OwnerUserID: 1,
		Result:      &entity.JobResult{UserID: 10},
	}
	if err := jobSvc.Save(context.Background(), &succeeded); err != nil {
		t.Fatalf("Save succeeded job: %v", err)
	}

	if err := pool.Start(context.Background(), 0); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := stopPoolWithTimeout(pool, time.Second); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	outcome, ok := pool.LastStopOutcome()
	if !ok {
		t.Fatal("LastStopOutcome missing")
	}
	if !outcome.WorkersStopped || !outcome.RepairAttempted {
		t.Fatalf("outcome=%+v, want stopped and repair attempted", outcome)
	}
	if outcome.RepairedActiveJobs != 1 {
		t.Fatalf("RepairedActiveJobs=%d want=1", outcome.RepairedActiveJobs)
	}
	if jobRepo.failActiveCalls.Load() != 1 {
		t.Fatalf("FailActive calls=%d want=1", jobRepo.failActiveCalls.Load())
	}

	if err := stopPoolWithTimeout(pool, time.Second); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
	secondOutcome, ok := pool.LastStopOutcome()
	if !ok {
		t.Fatal("second LastStopOutcome missing")
	}
	if secondOutcome != outcome {
		t.Fatalf("second outcome=%+v want same %+v", secondOutcome, outcome)
	}
	if jobRepo.failActiveCalls.Load() != 1 {
		t.Fatalf("second Stop ran repair again; FailActive calls=%d want=1", jobRepo.failActiveCalls.Load())
	}

	gotQueued, err := jobSvc.GetByID(context.Background(), queued.ID)
	if err != nil {
		t.Fatalf("GetByID queued: %v", err)
	}
	if gotQueued.Status != entity.JobFailed {
		t.Fatalf("queued status=%s want=%s", gotQueued.Status, entity.JobFailed)
	}
	gotSucceeded, err := jobSvc.GetByID(context.Background(), succeeded.ID)
	if err != nil {
		t.Fatalf("GetByID succeeded: %v", err)
	}
	if gotSucceeded.Status != entity.JobSucceeded {
		t.Fatalf("terminal status=%s want=%s", gotSucceeded.Status, entity.JobSucceeded)
	}
}

func TestStopUsesDetachedBoundedRepairContext(t *testing.T) {
	jobRepo := newBlockingRepairJobRepo()
	pool, _, _ := newTestPoolWithJobRepo(userrepo.NewMemoryUserRepository(), jobRepo)
	pool.repairTimeout = 20 * time.Millisecond

	if err := pool.Start(context.Background(), 0); err != nil {
		t.Fatalf("Start: %v", err)
	}

	stopCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := pool.Stop(stopCtx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Stop error=%v, want repair DeadlineExceeded", err)
	}
	if errors.Is(err, context.Canceled) {
		t.Fatalf("Stop returned caller cancellation instead of detached repair outcome: %v", err)
	}
	if jobRepo.failActiveCalls.Load() != 1 {
		t.Fatalf("FailActive calls=%d want=1", jobRepo.failActiveCalls.Load())
	}
	if jobRepo.sawCanceledAtStart.Load() {
		t.Fatal("repair context was already canceled by Stop caller context")
	}

	outcome, ok := pool.LastStopOutcome()
	if !ok {
		t.Fatal("LastStopOutcome missing")
	}
	if !outcome.RepairAttempted {
		t.Fatalf("outcome=%+v, want repair attempted", outcome)
	}
	if !errors.Is(outcome.RepairErr, context.DeadlineExceeded) {
		t.Fatalf("RepairErr=%v, want DeadlineExceeded", outcome.RepairErr)
	}
}

func TestMarkJobFailedUsesDetachedBoundedRepairContext(t *testing.T) {
	jobRepo := newBlockingSetFailedJobRepo()
	pool, _, jobSvc := newTestPoolWithJobRepo(userrepo.NewMemoryUserRepository(), jobRepo)
	pool.repairTimeout = 20 * time.Millisecond

	job := entity.Job{Status: entity.JobRunning, OwnerUserID: 1}
	if err := jobSvc.Save(context.Background(), &job); err != nil {
		t.Fatalf("Save job: %v", err)
	}

	parentCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := pool.markJobFailed(parentCtx, job.ID, service.ShutdownJobProblem())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("markJobFailed error=%v, want repair DeadlineExceeded", err)
	}
	if errors.Is(err, context.Canceled) {
		t.Fatalf("markJobFailed returned caller cancellation instead of detached repair outcome: %v", err)
	}
	if jobRepo.setFailedCalls.Load() != 1 {
		t.Fatalf("SetFailed calls=%d want=1", jobRepo.setFailedCalls.Load())
	}
	if jobRepo.sawCanceledAtStart.Load() {
		t.Fatal("terminal repair context was already canceled by worker context")
	}
}

func TestTimedOutStopDoesNotAccumulateWaiterGoroutines(t *testing.T) {
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
			Email: "goroutine-stop@example.com",
			Age:   21,
		},
	}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	waitClosed(t, userRepo.saveStarted, "worker to enter blocking Save")

	baseline := runtime.NumGoroutine()
	for i := 0; i < 20; i++ {
		stopCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		err := pool.Stop(stopCtx)
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Stop #%d error=%v, want DeadlineExceeded", i, err)
		}
	}

	assertGoroutinesBelow(t, baseline+5)

	close(userRepo.release)
	if err := stopPoolWithTimeout(pool, time.Second); err != nil {
		t.Fatalf("final Stop: %v", err)
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
	return newTestPoolWithServices(userRepo, jobSvc)
}

func newTestPoolWithJobRepo(userRepo service.UserRepository, jobRepo service.JobRepository) (*WorkerPool, *queue.Queue, *service.JobService) {
	jobSvc := service.NewJobService(jobRepo)
	return newTestPoolWithServices(userRepo, jobSvc)
}

func newTestPoolWithServices(userRepo service.UserRepository, jobSvc *service.JobService) (*WorkerPool, *queue.Queue, *service.JobService) {
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

func assertGoroutinesBelow(t *testing.T, limit int) {
	t.Helper()

	deadline := time.Now().Add(200 * time.Millisecond)
	for {
		if got := runtime.NumGoroutine(); got <= limit {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("goroutines=%d, want <=%d", runtime.NumGoroutine(), limit)
		}
		runtime.Gosched()
		time.Sleep(time.Millisecond)
	}
}

func stopPoolWithTimeout(pool *WorkerPool, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return pool.Stop(ctx)
}

type countingJobRepo struct {
	*jobrepo.MemoryJobRepository
	failActiveCalls atomic.Int32
}

func newCountingJobRepo() *countingJobRepo {
	return &countingJobRepo{MemoryJobRepository: jobrepo.NewMemoryJobRepository()}
}

func (r *countingJobRepo) FailActive(ctx context.Context, p entity.JobProblem) (int, error) {
	r.failActiveCalls.Add(1)
	return r.MemoryJobRepository.FailActive(ctx, p)
}

type blockingRepairJobRepo struct {
	*jobrepo.MemoryJobRepository
	failActiveCalls    atomic.Int32
	sawCanceledAtStart atomic.Bool
}

func newBlockingRepairJobRepo() *blockingRepairJobRepo {
	return &blockingRepairJobRepo{MemoryJobRepository: jobrepo.NewMemoryJobRepository()}
}

func (r *blockingRepairJobRepo) FailActive(ctx context.Context, p entity.JobProblem) (int, error) {
	r.failActiveCalls.Add(1)
	if ctx.Err() != nil {
		r.sawCanceledAtStart.Store(true)
	}
	<-ctx.Done()
	return 0, ctx.Err()
}

type blockingSetFailedJobRepo struct {
	*jobrepo.MemoryJobRepository
	setFailedCalls     atomic.Int32
	sawCanceledAtStart atomic.Bool
}

func newBlockingSetFailedJobRepo() *blockingSetFailedJobRepo {
	return &blockingSetFailedJobRepo{MemoryJobRepository: jobrepo.NewMemoryJobRepository()}
}

func (r *blockingSetFailedJobRepo) SetFailed(ctx context.Context, id int64, p entity.JobProblem) error {
	r.setFailedCalls.Add(1)
	if ctx.Err() != nil {
		r.sawCanceledAtStart.Store(true)
	}
	<-ctx.Done()
	return ctx.Err()
}
