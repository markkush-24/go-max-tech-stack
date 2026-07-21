package queue

import (
	"context"
	"errors"
	"pet-study/internal/entity"
	"testing"
	"time"
)

func TestEnqueueRejectsAlreadyCanceledContextBeforeSend(t *testing.T) {
	q := New(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := q.Enqueue(ctx, WorkItem{
		JobID: 1,
		Payload: entity.CreateUserInput{
			Name:  "canceled",
			Email: "canceled@example.com",
			Age:   21,
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Enqueue error=%v, want context.Canceled", err)
	}

	select {
	case item := <-q.Chan():
		t.Fatalf("canceled Enqueue sent item: %+v", item)
	default:
	}
}

func TestStopAcceptingIsPostReturnAdmissionBarrier(t *testing.T) {
	q := New(1)

	if err := q.Enqueue(context.Background(), WorkItem{
		JobID: 1,
		Payload: entity.CreateUserInput{
			Name:  "accepted",
			Email: "accepted@example.com",
			Age:   21,
		},
	}); err != nil {
		t.Fatalf("Enqueue before StopAccepting: %v", err)
	}

	q.StopAccepting()

	err := q.Enqueue(context.Background(), WorkItem{
		JobID: 2,
		Payload: entity.CreateUserInput{
			Name:  "stopped",
			Email: "stopped@example.com",
			Age:   22,
		},
	})
	if !errors.Is(err, ErrQueueStopped) {
		t.Fatalf("Enqueue after StopAccepting error=%v, want ErrQueueStopped", err)
	}

	got := <-q.Chan()
	if got.JobID != 1 {
		t.Fatalf("drained JobID=%d want=1", got.JobID)
	}

	select {
	case _, ok := <-q.Chan():
		if !ok {
			t.Fatal("StopAccepting closed producer channel")
		}
		t.Fatal("unexpected item after draining accepted work")
	default:
	}
}

func TestStopAcceptingWaitsForEnteredAdmissionBeforeReturning(t *testing.T) {
	q := New(1)

	q.mu.RLock()
	stopDone := make(chan struct{})
	go func() {
		q.StopAccepting()
		close(stopDone)
	}()

	select {
	case <-stopDone:
		t.Fatal("StopAccepting returned while admission barrier was held")
	default:
	}

	q.mu.RUnlock()

	select {
	case <-stopDone:
	case <-time.After(time.Second):
		t.Fatal("StopAccepting did not return after admission barrier released")
	}
}
