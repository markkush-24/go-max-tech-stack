package entity

import "testing"

func TestJobStateMachineAllowedTransitions(t *testing.T) {
	tests := []struct {
		name   string
		from   JobStatus
		intent JobTransition
		want   bool
	}{
		{name: "queued starts", from: JobQueued, intent: JobTransitionStart, want: true},
		{name: "queued fails", from: JobQueued, intent: JobTransitionFail, want: true},
		{name: "running succeeds", from: JobRunning, intent: JobTransitionSucceed, want: true},
		{name: "running fails", from: JobRunning, intent: JobTransitionFail, want: true},
		{name: "queued cannot succeed", from: JobQueued, intent: JobTransitionSucceed, want: false},
		{name: "running cannot start again", from: JobRunning, intent: JobTransitionStart, want: false},
		{name: "succeeded cannot fail", from: JobSucceeded, intent: JobTransitionFail, want: false},
		{name: "failed cannot start", from: JobFailed, intent: JobTransitionStart, want: false},
		{name: "unknown intent", from: JobQueued, intent: JobTransition("unknown"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanTransitionJob(tt.from, tt.intent); got != tt.want {
				t.Fatalf("CanTransitionJob(%q, %q)=%v want=%v", tt.from, tt.intent, got, tt.want)
			}
		})
	}
}

func TestJobTerminalStatuses(t *testing.T) {
	for _, status := range []JobStatus{JobSucceeded, JobFailed} {
		t.Run(string(status), func(t *testing.T) {
			if !status.IsTerminal() {
				t.Fatalf("%q must be terminal", status)
			}
			for _, intent := range []JobTransition{
				JobTransitionStart,
				JobTransitionSucceed,
				JobTransitionFail,
			} {
				if CanTransitionJob(status, intent) {
					t.Fatalf("terminal status %q must not allow intent %q", status, intent)
				}
			}
		})
	}
}
