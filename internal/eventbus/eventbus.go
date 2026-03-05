package eventbus

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sync"
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

func (e BaseEvent) GetType() EventType    { return e.Type }
func (e BaseEvent) GetTimestamp() int64   { return e.Timestamp }
func (e BaseEvent) GetSource() string     { return e.Source }

// EventHandler is a function that handles events
type EventHandler func(ctx context.Context, event Event) error

// Subscription represents an event subscription
type Subscription struct {
	ID        string
	EventType EventType
	Handler   EventHandler
}

// EventBus is the central event distribution system
type EventBus struct {
	handlers map[EventType][]*Subscription
	mu       sync.RWMutex
	wg       sync.WaitGroup
}

// New creates a new EventBus instance
func New() *EventBus {
	return &EventBus{
		handlers: make(map[EventType][]*Subscription),
	}
}

// Subscribe registers a handler for a specific event type
// Returns a subscription ID that can be used to unsubscribe
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) string {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subID := generateSubscriptionID(eventType)
	sub := &Subscription{
		ID:        subID,
		EventType: eventType,
		Handler:   handler,
	}

	eb.handlers[eventType] = append(eb.handlers[eventType], sub)
	log.Printf("[EventBus] Subscribed %s to event type %s", subID, eventType)
	return subID
}

// SubscribeAsync registers an async handler that runs in a goroutine
func (eb *EventBus) SubscribeAsync(eventType EventType, handler EventHandler) string {
	wrappedHandler := func(ctx context.Context, event Event) error {
		eb.wg.Add(1)
		go func() {
			defer eb.wg.Done()
			if err := handler(ctx, event); err != nil {
				log.Printf("[EventBus] Async handler error for %s: %v", eventType, err)
			}
		}()
		return nil
	}
	return eb.Subscribe(eventType, wrappedHandler)
}

// Unsubscribe removes a subscription by ID
func (eb *EventBus) Unsubscribe(subscriptionID string) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for eventType, subs := range eb.handlers {
		for i, sub := range subs {
			if sub.ID == subscriptionID {
				eb.handlers[eventType] = append(subs[:i], subs[i+1:]...)
				log.Printf("[EventBus] Unsubscribed %s from %s", subscriptionID, eventType)
				return nil
			}
		}
	}
	return fmt.Errorf("subscription %s not found", subscriptionID)
}

// Publish distributes an event to all registered handlers
func (eb *EventBus) Publish(ctx context.Context, event Event) {
	eb.mu.RLock()
	handlers := eb.handlers[event.GetType()]
	eb.mu.RUnlock()

	if len(handlers) == 0 {
		return
	}

	for _, sub := range handlers {
		go func(s *Subscription) {
			if err := s.Handler(ctx, event); err != nil {
				log.Printf("[EventBus] Handler error for %s: %v", s.EventType, err)
			}
		}(sub)
	}
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

// Wait waits for all async handlers to complete
func (eb *EventBus) Wait() {
	eb.wg.Wait()
}

// GetSubscribersCount returns the number of subscribers for an event type
func (eb *EventBus) GetSubscribersCount(eventType EventType) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.handlers[eventType])
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
