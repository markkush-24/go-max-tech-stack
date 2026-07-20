package entity

type JobStatus string

const (
	JobQueued    JobStatus = "queued"
	JobRunning   JobStatus = "running"
	JobSucceeded JobStatus = "succeeded"
	JobFailed    JobStatus = "failed"
)

type JobTransition string

const (
	JobTransitionStart   JobTransition = "start"
	JobTransitionSucceed JobTransition = "succeed"
	JobTransitionFail    JobTransition = "fail"
)

type JobTransitionSpec struct {
	Intent JobTransition
	From   []JobStatus
	To     JobStatus
}

var jobTransitionSpecs = map[JobTransition]JobTransitionSpec{
	JobTransitionStart: {
		Intent: JobTransitionStart,
		From:   []JobStatus{JobQueued},
		To:     JobRunning,
	},
	JobTransitionSucceed: {
		Intent: JobTransitionSucceed,
		From:   []JobStatus{JobRunning},
		To:     JobSucceeded,
	},
	JobTransitionFail: {
		Intent: JobTransitionFail,
		From:   []JobStatus{JobQueued, JobRunning},
		To:     JobFailed,
	},
}

func (s JobStatus) IsTerminal() bool {
	return s == JobSucceeded || s == JobFailed
}

func JobTransitionFor(intent JobTransition) (JobTransitionSpec, bool) {
	spec, ok := jobTransitionSpecs[intent]
	if ok {
		spec.From = append([]JobStatus(nil), spec.From...)
	}
	return spec, ok
}

func CanTransitionJob(from JobStatus, intent JobTransition) bool {
	spec, ok := JobTransitionFor(intent)
	if !ok {
		return false
	}
	for _, allowedFrom := range spec.From {
		if from == allowedFrom {
			return true
		}
	}
	return false
}

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
