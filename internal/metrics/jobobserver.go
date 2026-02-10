package metrics

import "time"

type JobsObserver interface {
	IncQueued()
	IncRunning()
	IncSucceeded()
	IncFailed()
	ObserveProcessing(d time.Duration)
}
