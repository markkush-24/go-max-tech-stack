package stream

import (
	"testing"
	"time"
)

func TestHubSubscribeReplaysRecentEventsInPublishOrder(t *testing.T) {
	hub := NewHub(3)

	hub.Publish(7, Event{Type: "queued", JobID: 7})
	hub.Publish(7, Event{Type: "running", JobID: 7})
	hub.Publish(7, Event{Type: "succeeded", JobID: 7})

	subscription, unsubscribe, err := hub.Subscribe(7)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer unsubscribe()

	want := []string{"queued", "running", "succeeded"}
	for i, wantType := range want {
		select {
		case ev := <-subscription.C:
			if ev.Type != wantType {
				t.Fatalf("event[%d]=%q want=%q", i, ev.Type, wantType)
			}
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for replayed event %q", wantType)
		}
	}
}

func TestHubPublishDoesNotBlockWhenSubscriberBufferIsFull(t *testing.T) {
	hub := NewHub(1)

	subscription, unsubscribe, err := hub.Subscribe(7)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer unsubscribe()

	hub.Publish(7, Event{Type: "queued", JobID: 7})

	done := make(chan struct{})
	go func() {
		hub.Publish(7, Event{Type: "running", JobID: 7})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked on a full subscriber buffer")
	}

	if got := (<-subscription.C).Type; got != "queued" {
		t.Fatalf("buffered event=%q want=%q", got, "queued")
	}
}
