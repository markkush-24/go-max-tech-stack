package jobrepo

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"pet-study/internal/apperr"
	"pet-study/internal/db"
	"pet-study/internal/entity"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

type casJobRepository interface {
	Save(context.Context, *entity.Job) error
	GetByID(context.Context, int64) (*entity.Job, error)
	SetRunning(context.Context, int64) error
	SetSucceeded(context.Context, int64, entity.JobResult) error
	SetFailed(context.Context, int64, entity.JobProblem) error
	FailActive(context.Context, entity.JobProblem) (int, error)
}

func TestJobRepositoryCASCompetingTerminalTransitions(t *testing.T) {
	for _, factory := range casRepositoryFactories() {
		t.Run(factory.name, func(t *testing.T) {
			repo := factory.new(t)
			ctx := context.Background()

			job := entity.Job{Status: entity.JobRunning, OwnerUserID: 1}
			if err := repo.Save(ctx, &job); err != nil {
				t.Fatalf("Save: %v", err)
			}

			start := make(chan struct{})
			errs := make(chan error, 2)
			go func() {
				<-start
				errs <- repo.SetSucceeded(ctx, job.ID, entity.JobResult{UserID: 42})
			}()
			go func() {
				<-start
				errs <- repo.SetFailed(ctx, job.ID, entity.JobProblem{Detail: "failed"})
			}()
			close(start)

			var successes int
			var conflicts int
			for i := 0; i < 2; i++ {
				err := <-errs
				switch {
				case err == nil:
					successes++
				case errors.Is(err, apperr.ErrJobTransitionConflict):
					conflicts++
				default:
					t.Fatalf("unexpected transition error=%v", err)
				}
			}

			if successes != 1 || conflicts != 1 {
				t.Fatalf("successes=%d conflicts=%d, want 1 and 1", successes, conflicts)
			}

			got, err := repo.GetByID(ctx, job.ID)
			if err != nil {
				t.Fatalf("GetByID: %v", err)
			}
			if !got.Status.IsTerminal() {
				t.Fatalf("final status=%q must be terminal", got.Status)
			}
		})
	}
}

func TestJobRepositoryCASTerminalStatesAreImmutable(t *testing.T) {
	for _, factory := range casRepositoryFactories() {
		t.Run(factory.name, func(t *testing.T) {
			tests := []struct {
				name   string
				status entity.JobStatus
				call   func(context.Context, casJobRepository, int64) error
			}{
				{
					name:   "failed cannot succeed",
					status: entity.JobFailed,
					call: func(ctx context.Context, repo casJobRepository, id int64) error {
						return repo.SetSucceeded(ctx, id, entity.JobResult{UserID: 42})
					},
				},
				{
					name:   "succeeded cannot fail",
					status: entity.JobSucceeded,
					call: func(ctx context.Context, repo casJobRepository, id int64) error {
						return repo.SetFailed(ctx, id, entity.JobProblem{Detail: "too late"})
					},
				},
				{
					name:   "succeeded cannot restart",
					status: entity.JobSucceeded,
					call: func(ctx context.Context, repo casJobRepository, id int64) error {
						return repo.SetRunning(ctx, id)
					},
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					repo := factory.new(t)
					ctx := context.Background()

					job := entity.Job{Status: tt.status, OwnerUserID: 1}
					if tt.status == entity.JobSucceeded {
						job.Result = &entity.JobResult{UserID: 7}
					}
					if tt.status == entity.JobFailed {
						job.Error = &entity.JobProblem{Detail: "already failed"}
					}
					if err := repo.Save(ctx, &job); err != nil {
						t.Fatalf("Save: %v", err)
					}

					err := tt.call(ctx, repo, job.ID)
					if !errors.Is(err, apperr.ErrJobTransitionConflict) {
						t.Fatalf("error=%v, want ErrJobTransitionConflict", err)
					}

					got, err := repo.GetByID(ctx, job.ID)
					if err != nil {
						t.Fatalf("GetByID: %v", err)
					}
					if got.Status != tt.status {
						t.Fatalf("status=%q want unchanged %q", got.Status, tt.status)
					}
				})
			}
		})
	}
}

func TestJobRepositoryCASDistinguishesNotFoundFromConflict(t *testing.T) {
	for _, factory := range casRepositoryFactories() {
		t.Run(factory.name, func(t *testing.T) {
			repo := factory.new(t)
			ctx := context.Background()

			if err := repo.SetRunning(ctx, 999); !errors.Is(err, entity.ErrJobNotFound) {
				t.Fatalf("missing job SetRunning error=%v, want ErrJobNotFound", err)
			}

			job := entity.Job{Status: entity.JobFailed, OwnerUserID: 1}
			if err := repo.Save(ctx, &job); err != nil {
				t.Fatalf("Save: %v", err)
			}

			err := repo.SetRunning(ctx, job.ID)
			if !errors.Is(err, apperr.ErrJobTransitionConflict) {
				t.Fatalf("terminal job SetRunning error=%v, want ErrJobTransitionConflict", err)
			}
			if errors.Is(err, entity.ErrJobNotFound) {
				t.Fatalf("transition conflict must not wrap ErrJobNotFound: %v", err)
			}
		})
	}
}

func TestJobRepositoryCASFailActiveLeavesTerminalJobsImmutable(t *testing.T) {
	for _, factory := range casRepositoryFactories() {
		t.Run(factory.name, func(t *testing.T) {
			repo := factory.new(t)
			ctx := context.Background()

			jobs := []entity.Job{
				{Status: entity.JobQueued, OwnerUserID: 1},
				{Status: entity.JobRunning, OwnerUserID: 1},
				{Status: entity.JobSucceeded, OwnerUserID: 1, Result: &entity.JobResult{UserID: 10}},
				{Status: entity.JobFailed, OwnerUserID: 1, Error: &entity.JobProblem{Detail: "already failed"}},
			}
			for i := range jobs {
				if err := repo.Save(ctx, &jobs[i]); err != nil {
					t.Fatalf("Save job %d: %v", i, err)
				}
			}

			n, err := repo.FailActive(ctx, entity.JobProblem{Detail: "shutdown"})
			if err != nil {
				t.Fatalf("FailActive: %v", err)
			}
			if n != 2 {
				t.Fatalf("FailActive count=%d want=2", n)
			}

			for _, original := range jobs {
				got, err := repo.GetByID(ctx, original.ID)
				if err != nil {
					t.Fatalf("GetByID(%d): %v", original.ID, err)
				}
				switch original.Status {
				case entity.JobQueued, entity.JobRunning:
					if got.Status != entity.JobFailed {
						t.Fatalf("job %d status=%q want failed", original.ID, got.Status)
					}
				case entity.JobSucceeded, entity.JobFailed:
					if got.Status != original.Status {
						t.Fatalf("terminal job %d status=%q want unchanged %q", original.ID, got.Status, original.Status)
					}
				}
			}
		})
	}
}

type casRepositoryFactory struct {
	name string
	new  func(*testing.T) casJobRepository
}

func casRepositoryFactories() []casRepositoryFactory {
	return []casRepositoryFactory{
		{
			name: "memory",
			new: func(t *testing.T) casJobRepository {
				t.Helper()
				return NewMemoryJobRepository()
			},
		},
		{
			name: "sqlx",
			new: func(t *testing.T) casJobRepository {
				t.Helper()
				state := newFakeJobSQLState()
				sqlDB := sql.OpenDB(fakeJobConnector{state: state})
				t.Cleanup(func() {
					_ = sqlDB.Close()
				})
				return NewSQLX(&db.DB{DB: sqlx.NewDb(sqlDB, "pgx")}, time.Second)
			},
		},
	}
}

type fakeJobSQLState struct {
	mu     sync.Mutex
	nextID int64
	jobs   map[int64]entity.Job
}

func newFakeJobSQLState() *fakeJobSQLState {
	return &fakeJobSQLState{
		nextID: 1,
		jobs:   make(map[int64]entity.Job),
	}
}

type fakeJobConnector struct {
	state *fakeJobSQLState
}

func (c fakeJobConnector) Connect(context.Context) (driver.Conn, error) {
	return fakeJobConn{state: c.state}, nil
}

func (c fakeJobConnector) Driver() driver.Driver {
	return fakeJobDriver{}
}

type fakeJobDriver struct{}

func (fakeJobDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("fakeJobDriver must be opened via connector")
}

type fakeJobConn struct {
	state *fakeJobSQLState
}

func (c fakeJobConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepared statements are not supported")
}

func (c fakeJobConn) Close() error {
	return nil
}

func (c fakeJobConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}

func (c fakeJobConn) CheckNamedValue(*driver.NamedValue) error {
	return nil
}

func (c fakeJobConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()

	switch normalizedSQL(query) {
	case normalizedSQL(qUpdateJob):
		id := int64Value(args[4].Value)
		job, ok := c.state.jobs[id]
		if !ok {
			return driver.RowsAffected(0), nil
		}
		job.Status = statusValue(args[0].Value)
		job.OwnerUserID = int64Value(args[1].Value)
		job.Result = resultFromValue(args[2].Value)
		job.Error = problemFromValue(args[3].Value)
		c.state.jobs[id] = cloneJob(job)
		return driver.RowsAffected(1), nil

	case normalizedSQL(qDeleteJob):
		id := int64Value(args[0].Value)
		if _, ok := c.state.jobs[id]; !ok {
			return driver.RowsAffected(0), nil
		}
		delete(c.state.jobs, id)
		return driver.RowsAffected(1), nil

	case normalizedSQL(qSetJobRunning):
		id := int64Value(args[1].Value)
		expected := statusValue(args[2].Value)
		job, ok := c.state.jobs[id]
		if !ok || job.Status != expected {
			return driver.RowsAffected(0), nil
		}
		job.Status = statusValue(args[0].Value)
		job.Result = nil
		job.Error = nil
		c.state.jobs[id] = cloneJob(job)
		return driver.RowsAffected(1), nil

	case normalizedSQL(qSetJobSucceeded):
		id := int64Value(args[2].Value)
		expected := statusValue(args[3].Value)
		job, ok := c.state.jobs[id]
		if !ok || job.Status != expected {
			return driver.RowsAffected(0), nil
		}
		job.Status = statusValue(args[0].Value)
		job.Result = resultFromValue(args[1].Value)
		job.Error = nil
		c.state.jobs[id] = cloneJob(job)
		return driver.RowsAffected(1), nil

	case normalizedSQL(qSetJobFailed):
		id := int64Value(args[2].Value)
		expectedA := statusValue(args[3].Value)
		expectedB := statusValue(args[4].Value)
		job, ok := c.state.jobs[id]
		if !ok || (job.Status != expectedA && job.Status != expectedB) {
			return driver.RowsAffected(0), nil
		}
		job.Status = statusValue(args[0].Value)
		job.Result = nil
		job.Error = problemFromValue(args[1].Value)
		c.state.jobs[id] = cloneJob(job)
		return driver.RowsAffected(1), nil

	case normalizedSQL(qFailActiveJobs):
		expectedA := statusValue(args[2].Value)
		expectedB := statusValue(args[3].Value)
		var changed int64
		for id, job := range c.state.jobs {
			if job.Status != expectedA && job.Status != expectedB {
				continue
			}
			job.Status = statusValue(args[0].Value)
			job.Result = nil
			job.Error = problemFromValue(args[1].Value)
			c.state.jobs[id] = cloneJob(job)
			changed++
		}
		return driver.RowsAffected(changed), nil
	}

	return nil, errors.New("unexpected exec query: " + query)
}

func (c fakeJobConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()

	switch normalizedSQL(query) {
	case normalizedSQL(qInsertJob):
		id := c.state.nextID
		c.state.nextID++
		job := entity.Job{
			ID:          id,
			Status:      statusValue(args[0].Value),
			OwnerUserID: int64Value(args[1].Value),
			Result:      resultFromValue(args[2].Value),
			Error:       problemFromValue(args[3].Value),
		}
		c.state.jobs[id] = cloneJob(job)
		return &fakeRows{
			columns: []string{"id"},
			values:  [][]driver.Value{{id}},
		}, nil

	case normalizedSQL(qGetJobById):
		id := int64Value(args[0].Value)
		job, ok := c.state.jobs[id]
		if !ok {
			return &fakeRows{columns: []string{"id", "status", "owner_user_id", "result_user_id", "error_payload"}}, nil
		}
		return &fakeRows{
			columns: []string{"id", "status", "owner_user_id", "result_user_id", "error_payload"},
			values:  [][]driver.Value{jobRowValues(job)},
		}, nil

	case normalizedSQL(qGetJobStatus):
		id := int64Value(args[0].Value)
		job, ok := c.state.jobs[id]
		if !ok {
			return &fakeRows{columns: []string{"status"}}, nil
		}
		return &fakeRows{
			columns: []string{"status"},
			values:  [][]driver.Value{{string(job.Status)}},
		}, nil

	case normalizedSQL(qGetAllJobs):
		rows := &fakeRows{columns: []string{"id", "status", "owner_user_id", "result_user_id", "error_payload"}}
		for _, job := range c.state.jobs {
			rows.values = append(rows.values, jobRowValues(job))
		}
		return rows, nil
	}

	return nil, errors.New("unexpected query: " + query)
}

type fakeRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r *fakeRows) Columns() []string {
	return r.columns
}

func (r *fakeRows) Close() error {
	return nil
}

func (r *fakeRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

func normalizedSQL(query string) string {
	return strings.Join(strings.Fields(query), " ")
}

func int64Value(value any) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case int32:
		return int64(v)
	default:
		panic("unexpected int64 value")
	}
}

func statusValue(value any) entity.JobStatus {
	switch v := value.(type) {
	case entity.JobStatus:
		return v
	case string:
		return entity.JobStatus(v)
	case []byte:
		return entity.JobStatus(string(v))
	default:
		panic("unexpected status value")
	}
}

func resultFromValue(value any) *entity.JobResult {
	if value == nil {
		return nil
	}
	if result, ok := value.(sql.NullInt64); ok {
		if !result.Valid {
			return nil
		}
		return &entity.JobResult{UserID: result.Int64}
	}
	result := entity.JobResult{UserID: int64Value(value)}
	return &result
}

func problemFromValue(value any) *entity.JobProblem {
	if value == nil {
		return nil
	}
	var payload []byte
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			return nil
		}
		payload = v
	case string:
		if v == "" {
			return nil
		}
		payload = []byte(v)
	default:
		panic("unexpected problem payload")
	}
	var problem entity.JobProblem
	if err := json.Unmarshal(payload, &problem); err != nil {
		panic(err)
	}
	return &problem
}

func jobRowValues(job entity.Job) []driver.Value {
	var result any
	if job.Result != nil {
		result = job.Result.UserID
	}
	var problem any
	if job.Error != nil {
		payload, err := json.Marshal(job.Error)
		if err != nil {
			panic(err)
		}
		problem = payload
	}
	return []driver.Value{
		job.ID,
		string(job.Status),
		job.OwnerUserID,
		result,
		problem,
	}
}
