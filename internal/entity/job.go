package entity

type JobStatus string

const (
	JobQueued    JobStatus = "queued"
	JobRunning   JobStatus = "running"
	JobSucceeded JobStatus = "succeeded"
	JobFailed    JobStatus = "failed"
)

type Job struct {
	ID     int64     `json:"id"`
	Status JobStatus `json:"status"`

	Result      *JobResult  `json:"result,omitempty"`
	Error       *JobProblem `json:"error,omitempty"`
	OwnerUserID int64       `json:"-"`
}

type JobResult struct {
	UserID int64 `json:"user_id"`
}

type JobProblem struct {
	Type   string `json:"type,omitempty"`
	Title  string `json:"title,omitempty"`
	Status int    `json:"status,omitempty"`
	Detail string `json:"detail,omitempty"`

	Instance string `json:"instance,omitempty"`

	RequestID     string            `json:"request_id,omitempty"`
	InvalidParams []JobInvalidParam `json:"invalid_params,omitempty"`
}

type JobInvalidParam struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type JobBridgeDTO struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
	Source string `json:"source"`
}
