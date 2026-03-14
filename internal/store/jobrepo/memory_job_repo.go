package jobrepo

import (
	"context"
	"pet-study/internal/entity"
	"sync"
)

func NewMemoryJobRepository() *MemoryJobRepository {
	return &MemoryJobRepository{
		jobs:   make(map[int64]*entity.Job),
		nextID: 1,
	}
}

// MemoryJobRepository — in-memory реализация job storage.
// Здесь нет реального I/O, поэтому ctx не участвует в ожиданиях/отмене,
// а используется только как часть общего контракта интерфейса.
type MemoryJobRepository struct {
	jobs   map[int64]*entity.Job
	mux    sync.RWMutex
	nextID int64
}

func (r *MemoryJobRepository) Save(ctx context.Context, job *entity.Job) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	if job.ID == 0 {
		job.ID = r.nextID
		r.nextID++
	}

	j := cloneJob(*job) // копия уже с корректным ID
	r.jobs[j.ID] = &j   // храним копию
	return nil
}

func (r *MemoryJobRepository) GetAll(ctx context.Context) ([]*entity.Job, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()

	arrJobs := make([]*entity.Job, 0, len(r.jobs))
	for _, j := range r.jobs {
		cp := cloneJob(*j)
		arrJobs = append(arrJobs, &cp)
	}
	return arrJobs, nil
}

func (r *MemoryJobRepository) GetByID(ctx context.Context, id int64) (*entity.Job, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()

	j, ok := r.jobs[id]
	if !ok {
		return nil, entity.ErrJobNotFound
	}
	cp := cloneJob(*j)
	return &cp, nil
}

func (r *MemoryJobRepository) Delete(ctx context.Context, id int64) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	if _, ok := r.jobs[id]; !ok {
		return entity.ErrJobNotFound
	}
	delete(r.jobs, id)
	return nil
}

func cloneJob(in entity.Job) entity.Job {
	out := in

	if in.Result != nil {
		r := *in.Result
		out.Result = &r
	}
	if in.Error != nil {
		p := *in.Error
		if in.Error.InvalidParams != nil {
			p.InvalidParams = append([]entity.JobInvalidParam(nil), in.Error.InvalidParams...)
		}
		out.Error = &p
	}
	return out
}

func (r *MemoryJobRepository) SetRunning(ctx context.Context, id int64) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	job, ok := r.jobs[id]
	if !ok {
		return entity.ErrJobNotFound
	}
	job.Status = entity.JobRunning
	job.Result = nil
	job.Error = nil
	return nil
}

func (r *MemoryJobRepository) SetSucceeded(ctx context.Context, id int64, res entity.JobResult) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	job, ok := r.jobs[id]
	if !ok {
		return entity.ErrJobNotFound
	}
	job.Status = entity.JobSucceeded
	rr := res
	job.Result = &rr
	job.Error = nil
	return nil
}

func (r *MemoryJobRepository) SetFailed(ctx context.Context, id int64, p entity.JobProblem) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	job, ok := r.jobs[id]
	if !ok {
		return entity.ErrJobNotFound
	}
	job.Status = entity.JobFailed
	pp := p
	if pp.InvalidParams != nil {
		pp.InvalidParams = append([]entity.JobInvalidParam(nil), pp.InvalidParams...)
	}
	job.Error = &pp
	job.Result = nil
	return nil
}

func (r *MemoryJobRepository) FailActive(ctx context.Context, p entity.JobProblem) (int, error) {
	r.mux.Lock()
	defer r.mux.Unlock()

	failed := 0
	for _, job := range r.jobs {
		if job.Status != entity.JobQueued && job.Status != entity.JobRunning {
			continue
		}

		job.Status = entity.JobFailed
		pp := p
		if pp.InvalidParams != nil {
			pp.InvalidParams = append([]entity.JobInvalidParam(nil), pp.InvalidParams...)
		}
		job.Error = &pp
		job.Result = nil
		failed++
	}

	return failed, nil
}
