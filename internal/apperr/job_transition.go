package apperr

import (
	"errors"
	"fmt"
	"pet-study/internal/entity"
)

var ErrJobTransitionConflict = errors.New("job transition conflict")

type JobTransitionConflictError struct {
	JobID  int64
	From   entity.JobStatus
	To     entity.JobStatus
	Intent entity.JobTransition
}

func (e *JobTransitionConflictError) Error() string {
	return fmt.Sprintf("job %d cannot transition from %q to %q with intent %q", e.JobID, e.From, e.To, e.Intent)
}

func (e *JobTransitionConflictError) Unwrap() []error {
	return []error{ErrJobTransitionConflict, ErrConflict}
}

func NewJobTransitionConflict(jobID int64, from entity.JobStatus, spec entity.JobTransitionSpec) error {
	return &JobTransitionConflictError{
		JobID:  jobID,
		From:   from,
		To:     spec.To,
		Intent: spec.Intent,
	}
}
