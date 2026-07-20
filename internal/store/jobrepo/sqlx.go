package jobrepo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"pet-study/internal/apperr"
	"pet-study/internal/db"
	"pet-study/internal/entity"
	"time"
)

const (
	qGetJobById = `
SELECT id, status, owner_user_id, result_user_id, error_payload
FROM jobs
WHERE id = $1
`
	qGetJobStatus = `
SELECT status
FROM jobs
WHERE id = $1
`
	qInsertJob = `
INSERT INTO jobs (status, owner_user_id, result_user_id, error_payload)
VALUES ($1, $2, $3,$4)
RETURNING id
`

	qUpdateJob = `
UPDATE jobs
SET
	status = $1,
	owner_user_id = $2,
	result_user_id = $3,
	error_payload = $4,
	updated_at = now()
WHERE id = $5
`

	qSetJobRunning = `
UPDATE jobs
SET
	status = $1,
	result_user_id = NULL,
	error_payload = NULL,
	started_at = now(),
	finished_at = NULL,
	updated_at = now()
WHERE id = $2 AND status = $3
`

	qSetJobSucceeded = `
UPDATE jobs
SET
	status = $1,
	result_user_id = $2,
	error_payload = NULL,
	finished_at = now(),
	updated_at = now()
WHERE id = $3 AND status = $4
`

	qSetJobFailed = `
UPDATE jobs
SET
	status = $1,
	result_user_id = NULL,
	error_payload = $2,
	finished_at = now(),
	updated_at = now()
WHERE id = $3 AND status IN ($4, $5)
`

	qFailActiveJobs = `
UPDATE jobs
SET
	status = $1,
	result_user_id = NULL,
	error_payload = $2,
	finished_at = now(),
	updated_at = now()
WHERE status IN ($3, $4)
`
	qGetAllJobs = `
SELECT id, status, owner_user_id, result_user_id, error_payload
FROM jobs
ORDER BY id
`

	qDeleteJob = `
DELETE FROM jobs
WHERE id = $1
`
)

type jobRow struct {
	ID           int64         `db:"id"`
	Status       string        `db:"status"`
	OwnerUserID  int64         `db:"owner_user_id"`
	ResultUserID sql.NullInt64 `db:"result_user_id"`
	ErrorPayload []byte        `db:"error_payload"`
}

type SQLXJobRepository struct {
	sqlDB        *db.DB
	queryTimeout time.Duration
}

func NewSQLX(sqlDB *db.DB, queryTimeout time.Duration) *SQLXJobRepository {
	return &SQLXJobRepository{
		sqlDB:        sqlDB,
		queryTimeout: queryTimeout,
	}
}

func (r *SQLXJobRepository) GetAll(ctx context.Context) ([]*entity.Job, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	rows, err := r.sqlDB.QueryContext(ctx, qGetAllJobs)
	if err != nil {
		return nil, mapJobRepoErr(err)
	}
	defer rows.Close()

	jobs := make([]*entity.Job, 0)

	for rows.Next() {
		var row jobRow
		if err := rows.Scan(
			&row.ID,
			&row.Status,
			&row.OwnerUserID,
			&row.ResultUserID,
			&row.ErrorPayload,
		); err != nil {
			return nil, err
		}

		job, err := toEntityJob(row)
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, mapJobRepoErr(err)
	}

	return jobs, nil
}

func (r *SQLXJobRepository) GetByID(ctx context.Context, id int64) (*entity.Job, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	var row jobRow
	err := r.sqlDB.GetContext(ctx, &row, qGetJobById, id)
	if err != nil {
		return nil, mapJobRepoErr(err)
	}

	job, err := toEntityJob(row)
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (r *SQLXJobRepository) Save(ctx context.Context, job *entity.Job) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	resultUserID := toNullInt64(job.Result)

	errorPayload, err := toJSONBytes(job.Error)
	if err != nil {
		return err
	}

	if job.ID == 0 {
		err := r.sqlDB.QueryRowContext(
			ctx,
			qInsertJob,
			job.Status,
			job.OwnerUserID,
			resultUserID,
			errorPayload,
		).Scan(&job.ID)
		if err != nil {
			return mapJobRepoErr(err)
		}
		return nil
	}

	res, err := r.sqlDB.ExecContext(
		ctx,
		qUpdateJob,
		job.Status,
		job.OwnerUserID,
		resultUserID,
		errorPayload,
		job.ID,
	)
	if err != nil {
		return mapJobRepoErr(err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return entity.ErrJobNotFound
	}

	return nil
}

func (r *SQLXJobRepository) Delete(ctx context.Context, id int64) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	res, err := r.sqlDB.ExecContext(ctx, qDeleteJob, id)
	if err != nil {
		return mapJobRepoErr(err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return entity.ErrJobNotFound
	}

	return nil
}

func (r *SQLXJobRepository) SetRunning(ctx context.Context, id int64) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	res, err := r.sqlDB.ExecContext(ctx, qSetJobRunning, entity.JobRunning, id, entity.JobQueued)
	if err != nil {
		return mapJobRepoErr(err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return r.transitionConflictOrNotFound(ctx, id, entity.JobTransitionStart)
	}

	return nil
}
func (r *SQLXJobRepository) SetSucceeded(ctx context.Context, id int64, res entity.JobResult) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	dbRes, err := r.sqlDB.ExecContext(
		ctx,
		qSetJobSucceeded,
		entity.JobSucceeded,
		res.UserID,
		id,
		entity.JobRunning,
	)
	if err != nil {
		return mapJobRepoErr(err)
	}

	n, err := dbRes.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return r.transitionConflictOrNotFound(ctx, id, entity.JobTransitionSucceed)
	}

	return nil
}
func (r *SQLXJobRepository) SetFailed(ctx context.Context, id int64, p entity.JobProblem) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	errorPayload, err := toJSONBytes(&p)
	if err != nil {
		return err
	}

	dbRes, err := r.sqlDB.ExecContext(
		ctx,
		qSetJobFailed,
		entity.JobFailed,
		errorPayload,
		id,
		entity.JobQueued,
		entity.JobRunning,
	)
	if err != nil {
		return mapJobRepoErr(err)
	}

	n, err := dbRes.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return r.transitionConflictOrNotFound(ctx, id, entity.JobTransitionFail)
	}

	return nil
}
func (r *SQLXJobRepository) FailActive(ctx context.Context, p entity.JobProblem) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	errorPayload, err := toJSONBytes(&p)
	if err != nil {
		return 0, err
	}

	dbRes, err := r.sqlDB.ExecContext(
		ctx,
		qFailActiveJobs,
		entity.JobFailed,
		errorPayload,
		entity.JobQueued,
		entity.JobRunning,
	)
	if err != nil {
		return 0, mapJobRepoErr(err)
	}

	n, err := dbRes.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(n), nil
}

func toEntityJob(row jobRow) (*entity.Job, error) {
	job := &entity.Job{
		ID:          row.ID,
		Status:      entity.JobStatus(row.Status),
		OwnerUserID: row.OwnerUserID,
	}

	if row.ResultUserID.Valid {
		job.Result = &entity.JobResult{
			UserID: row.ResultUserID.Int64,
		}
	}

	if len(row.ErrorPayload) > 0 {
		var p entity.JobProblem
		if err := json.Unmarshal(row.ErrorPayload, &p); err != nil {
			return nil, err
		}
		job.Error = &p
	}

	return job, nil
}

func mapJobRepoErr(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return entity.ErrJobNotFound
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	return err
}

func (r *SQLXJobRepository) transitionConflictOrNotFound(ctx context.Context, id int64, intent entity.JobTransition) error {
	var status string
	if err := r.sqlDB.GetContext(ctx, &status, qGetJobStatus, id); err != nil {
		return mapJobRepoErr(err)
	}

	spec, ok := entity.JobTransitionFor(intent)
	if !ok {
		return apperr.NewJobTransitionConflict(id, entity.JobStatus(status), entity.JobTransitionSpec{Intent: intent})
	}
	return apperr.NewJobTransitionConflict(id, entity.JobStatus(status), spec)
}

func toNullInt64(result *entity.JobResult) sql.NullInt64 {
	if result == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{
		Int64: result.UserID,
		Valid: true,
	}
}

func toJSONBytes(problem *entity.JobProblem) ([]byte, error) {
	if problem == nil {
		return nil, nil
	}
	return json.Marshal(problem)
}
