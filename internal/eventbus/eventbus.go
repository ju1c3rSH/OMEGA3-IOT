package eventbus

import (
	"context"
	"expvar"
	"fmt"
	"log"
	"reflect"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

// EventType represents the type of an event
type EventType string

// Event is the base interface for all events
type Event interface {
	GetType() EventType
	GetTimestamp() int64
	GetSource() string
}

// BaseEvent provides common fields for all events
type BaseEvent struct {
	Type      EventType `json:"type"`
	Timestamp int64     `json:"timestamp"`
	Source    string    `json:"source"`
}

func (e BaseEvent) GetType() EventType  { return e.Type }
func (e BaseEvent) GetTimestamp() int64 { return e.Timestamp }
func (e BaseEvent) GetSource() string   { return e.Source }

// EventHandler is a function that handles events
type EventHandler func(ctx context.Context, event Event) error

// defaultQueueSize is the bounded queue size per subscription (drop-newest)
const defaultQueueSize = 512

// droppedStats tracks dropped events per event type (visible at /debug/vars)
var droppedStats = expvar.NewMap("eventbus.dropped")

// dropLogCounts samples drop logs: one log line per 1000 drops per event type
var dropLogCounts sync.Map // map[EventType]*atomic.Int64

// envelope carries the publish-time context alongside the event
type envelope struct {
	ctx   context.Context
	event Event
}

// Subscription represents an event subscription with a dedicated dispatch goroutine
type Subscription struct {
	ID        string
	EventType EventType
	Handler   EventHandler

	// queue is never closed; dispatcher exits via done, so Publish never
	// sends on a closed channel
	queue chan envelope
	done  chan struct{}
}

// EventBus is the central event distribution system
type EventBus struct {
	handlers map[EventType][]*Subscription
	mu       sync.RWMutex
	wg       sync.WaitGroup
	stopped  bool
}

// New creates a new EventBus instance
func New() *EventBus {
	return &EventBus{
		handlers: make(map[EventType][]*Subscription),
	}
}

// Subscribe registers a handler for a specific event type and starts a
// dedicated dispatch goroutine with a bounded queue.
// Returns a subscription ID that can be used to unsubscribe.
// Returns "" if the bus is already stopped.
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) string {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.stopped {
		log.Printf("[EventBus] Subscribe %s ignored: bus stopped", eventType)
		return ""
	}

	subID := generateSubscriptionID(eventType)
	sub := &Subscription{
		ID:        subID,
		EventType: eventType,
		Handler:   handler,
		queue:     make(chan envelope, defaultQueueSize),
		done:      make(chan struct{}),
	}

	eb.handlers[eventType] = append(eb.handlers[eventType], sub)
	eb.wg.Add(1)
	go eb.dispatch(sub)
	log.Printf("[EventBus] Subscribed %s to event type %s", subID, eventType)
	return subID
}

// Unsubscribe removes a subscription by ID and stops its dispatch goroutine
func (eb *EventBus) Unsubscribe(subscriptionID string) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for eventType, subs := range eb.handlers {
		for i, sub := range subs {
			if sub.ID == subscriptionID {
				eb.handlers[eventType] = append(subs[:i], subs[i+1:]...)
				close(sub.done)
				log.Printf("[EventBus] Unsubscribed %s from %s", subscriptionID, eventType)
				return nil
			}
		}
	}
	return fmt.Errorf("subscription %s not found", subscriptionID)
}

// Publish distributes an event to all subscribers via non-blocking enqueue.
// If a subscriber's queue is full the event is dropped (drop-newest) and
// counted in the eventbus.dropped expvar map.
func (eb *EventBus) Publish(ctx context.Context, event Event) {
	eb.mu.RLock()
	subs := eb.handlers[event.GetType()]
	eb.mu.RUnlock()

	for _, sub := range subs {
		select {
		case sub.queue <- envelope{ctx: ctx, event: event}:
		default:
			droppedStats.Add(string(sub.EventType), 1)
			if n := nextDropLog(sub.EventType); n == 1 || n%1000 == 0 {
				log.Printf("[EventBus] queue full for %s, dropping events (dropped=%d total)", sub.EventType, n)
			}
		}
	}
}

// dispatch drains the subscription queue until done is closed
func (eb *EventBus) dispatch(sub *Subscription) {
	defer eb.wg.Done()
	for {
		select {
		case env := <-sub.queue:
			eb.invoke(sub, env)
		case <-sub.done:
			return
		}
	}
}

// invoke runs the handler detached from the publish-time context (which may
// already be cancelled, e.g. the 2s MQTT worker ctx) while preserving values
func (eb *EventBus) invoke(sub *Subscription, env envelope) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[EventBus] handler %s panic on %s: %v\n%s", sub.ID, sub.EventType, r, debug.Stack())
		}
	}()
	ctx := context.WithoutCancel(env.ctx)
	if err := sub.Handler(ctx, env.event); err != nil {
		log.Printf("[EventBus] Handler error for %s: %v", sub.EventType, err)
	}
}

// Stop closes all subscription queues' dispatch loops and waits for every
// dispatcher goroutine to exit. Idempotent; Publish after Stop just drops.
func (eb *EventBus) Stop() {
	eb.mu.Lock()
	if eb.stopped {
		eb.mu.Unlock()
		return
	}
	eb.stopped = true
	for _, subs := range eb.handlers {
		for _, sub := range subs {
			close(sub.done)
		}
	}
	eb.handlers = make(map[EventType][]*Subscription)
	wg := &eb.wg
	eb.mu.Unlock()

	wg.Wait()
	log.Printf("[EventBus] stopped, all dispatchers exited")
}

// PublishSync distributes an event synchronously
func (eb *EventBus) PublishSync(ctx context.Context, event Event) error {
	eb.mu.RLock()
	handlers := eb.handlers[event.GetType()]
	eb.mu.RUnlock()

	if len(handlers) == 0 {
		return nil
	}

	var errs []error
	for _, sub := range handlers {
		if err := sub.Handler(ctx, event); err != nil {
			errs = append(errs, fmt.Errorf("handler %s: %w", sub.ID, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("sync publish errors: %v", errs)
	}
	return nil
}

// GetSubscribersCount returns the number of subscribers for an event type
func (eb *EventBus) GetSubscribersCount(eventType EventType) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.handlers[eventType])
}

func nextDropLog(eventType EventType) int64 {
	c, _ := dropLogCounts.LoadOrStore(eventType, new(atomic.Int64))
	return c.(*atomic.Int64).Add(1)
}

// generateSubscriptionID creates a unique subscription ID
var subCounter int
var subMu sync.Mutex

func generateSubscriptionID(eventType EventType) string {
	subMu.Lock()
	defer subMu.Unlock()
	subCounter++
	return fmt.Sprintf("%s-%d", eventType, subCounter)
}

// TypedEventHandler is a type-safe wrapper for event handlers
type TypedEventHandler[T Event] func(ctx context.Context, event T) error

// SubscribeTyped registers a typed handler for a specific event type
func SubscribeTyped[T Event](eb *EventBus, eventType EventType, handler TypedEventHandler[T]) string {
	wrappedHandler := func(ctx context.Context, event Event) error {
		typedEvent, ok := event.(T)
		if !ok {
			return fmt.Errorf("event type mismatch: expected %s, got %s",
				reflect.TypeOf((*T)(nil)).Elem(), reflect.TypeOf(event))
		}
		return handler(ctx, typedEvent)
	}
	return eb.Subscribe(eventType, wrappedHandler)
}
