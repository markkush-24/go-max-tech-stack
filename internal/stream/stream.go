package stream

import (
	"context"
	"encoding/json"
	"errors"
	"expvar"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var ErrHubClosed = errors.New("stream hub is closed")

var (
	outOnce        sync.Once
	sseSubscribers *expvar.Map
	sseEventsTotal *expvar.Map
	sseDropsTotal  *expvar.Map
)

type Hub struct {
	mu               sync.Mutex
	subs             map[int64]map[uint64]*subscriber
	nextID           uint64
	subscriberBuffer int

	subscribers int64
	eventsTotal int64
	dropsTotal  int64
	closed      atomic.Bool
}

type Subscription struct {
	C <-chan Event
}

type Event struct {
	Type  string
	JobID int64
	At    time.Time
	Data  any
}

type subscriber struct {
	id    uint64
	jobID int64
	ch    chan Event
}

func NewHub(subscriberBuffer int) *Hub {
	outOnce.Do(func() {
		sseSubscribers = expvar.NewMap("sse_subscribers")
		sseEventsTotal = expvar.NewMap("sse_events_total")
		sseDropsTotal = expvar.NewMap("sse_drops_total")
	})

	if subscriberBuffer <= 0 {
		panic("subscriberBuffer must be > 0")
	}

	return &Hub{
		closed:           atomic.Bool{},
		subs:             make(map[int64]map[uint64]*subscriber),
		subscriberBuffer: subscriberBuffer,
	}
}

func (h *Hub) Subscribe(jobID int64) (Subscription, func(), error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.closed.Load() {
		h.nextID++
		id := h.nextID

		ch := make(chan Event, h.subscriberBuffer)

		if h.subs[jobID] == nil {
			h.subs[jobID] = make(map[uint64]*subscriber)
		}

		h.subs[jobID][id] = &subscriber{
			id:    id,
			jobID: jobID,
			ch:    ch,
		}

		h.subscribers++
		sseSubscribers.Add("subscribers", 1)

		var once sync.Once

		unsubscribe := func() {
			once.Do(func() {
				h.mu.Lock()
				defer h.mu.Unlock()

				jobSubs, ok := h.subs[jobID]
				if !ok {
					return
				}

				if _, ok := jobSubs[id]; !ok {
					return
				}

				delete(jobSubs, id)
				h.subscribers--
				sseSubscribers.Add("subscribers", -1)

				if len(jobSubs) == 0 {
					delete(h.subs, jobID)
				}
			})
		}

		return Subscription{C: ch}, unsubscribe, nil
	}
	return Subscription{}, func() {}, ErrHubClosed
}

func (h *Hub) Publish(jobID int64, ev Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.closed.Load() {
		h.eventsTotal++
		sseEventsTotal.Add("eventsTotal", 1)

		for _, sub := range h.subs[jobID] {
			select {
			case sub.ch <- ev:
			default:
				h.dropsTotal++
				sseDropsTotal.Add("dropsTotal", 1)
			}
		}
	}
}

func (h *Hub) Subscribers() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.subscribers
}

func (h *Hub) EventsTotal() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.eventsTotal
}

func (h *Hub) DropsTotal() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.dropsTotal
}

// WriteSSE writes a single event to the response writer
func WriteSSE(w http.ResponseWriter, event Event) error {
	// Write event type if specified
	if event.Type != "" {
		_, err := fmt.Fprintf(w, "event: %s\n", event.Type)
		if err != nil {
			return err
		}
	}

	// Marshal data to JSON
	data, err := json.Marshal(event.Data)
	if err != nil {
		return err
	}

	// Write data and blank line to end event
	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	if err != nil {
		return err
	}

	return nil
}

func (h *Hub) Ready(ctx context.Context) error {
	if h.closed.Load() {
		return ErrHubClosed
	}
	return nil
}

func (h *Hub) Close() error {
	if h == nil {
		return nil
	}

	if !h.closed.CompareAndSwap(false, true) {
		return nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for jobID, jobSubs := range h.subs {
		for id, sub := range jobSubs {
			close(sub.ch)
			delete(jobSubs, id)
		}
		delete(h.subs, jobID)
	}

	if h.subscribers != 0 {
		sseSubscribers.Add("subscribers", -h.subscribers)
		h.subscribers = 0
	}

	return nil
}
