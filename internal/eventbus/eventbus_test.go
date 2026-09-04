package eventbus

import (
	"context"
	"expvar"
	"runtime"
	"strconv"
	"testing"
	"time"
)

type testEvent struct {
	BaseEvent
}

func newTestEvent(t EventType) *testEvent {
	return &testEvent{BaseEvent{Type: t}}
}

func waitTimeout(ch <-chan struct{}, d time.Duration) bool {
	select {
	case <-ch:
		return true
	case <-time.After(d):
		return false
	}
}

func TestPublishDeliversEvent(t *testing.T) {
	eb := New()
	got := make(chan Event, 1)
	eb.Subscribe("t1", func(ctx context.Context, e Event) error {
		got <- e
		return nil
	})
	eb.Publish(context.Background(), newTestEvent("t1"))

	select {
	case e := <-got:
		if e.GetType() != "t1" {
			t.Fatalf("unexpected event type: %s", e.GetType())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not receive event")
	}
	eb.Stop()
}

func TestPublishAfterUnsubscribeDoesNotPanic(t *testing.T) {
	eb := New()
	id := eb.Subscribe("t2", func(ctx context.Context, e Event) error { return nil })
	if err := eb.Unsubscribe(id); err != nil {
		t.Fatalf("unsubscribe failed: %v", err)
	}
	if err := eb.Unsubscribe(id); err == nil {
		t.Fatal("double unsubscribe should return not-found error")
	}
	for i := 0; i < 2000; i++ {
		eb.Publish(context.Background(), newTestEvent("t2"))
	}
	eb.Stop()
	eb.Stop()
}

func TestDropCounterIncrements(t *testing.T) {
	eb := New()
	evtType := EventType("t3.drop")
	release := make(chan struct{})
	eb.Subscribe(evtType, func(ctx context.Context, e Event) error {
		<-release
		return nil
	})

	before := droppedCount(evtType)
	for i := 0; i < defaultQueueSize+5; i++ {
		eb.Publish(context.Background(), newTestEvent(evtType))
	}
	after := droppedCount(evtType)
	if after <= before {
		t.Fatalf("drop counter did not increase: before=%d after=%d", before, after)
	}
	close(release)
	eb.Stop()
}

func droppedCount(t EventType) int64 {
	var v int64
	droppedStats.Do(func(kv expvar.KeyValue) {
		if kv.Key == string(t) {
			n, err := strconv.ParseInt(kv.Value.String(), 10, 64)
			if err == nil {
				v = n
			}
		}
	})
	return v
}

func TestStopExitsDispatchersAndRejectsSubscribe(t *testing.T) {
	base := runtime.NumGoroutine()
	eb := New()
	handled := make(chan struct{})
	eb.Subscribe("t4", func(ctx context.Context, e Event) error {
		close(handled)
		return nil
	})
	eb.Publish(context.Background(), newTestEvent("t4"))
	if !waitTimeout(handled, 2*time.Second) {
		t.Fatal("handler not invoked before Stop")
	}
	eb.Stop()

	if id := eb.Subscribe("t5", func(ctx context.Context, e Event) error { return nil }); id != "" {
		t.Fatalf("expected empty sub id after Stop, got %q", id)
	}
	for i := 0; i < 100; i++ {
		eb.Publish(context.Background(), newTestEvent("t4"))
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= base {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("dispatcher goroutines still running: base=%d now=%d", base, runtime.NumGoroutine())
}

func TestPanicInHandlerDoesNotKillDispatcher(t *testing.T) {
	eb := New()
	calls := make(chan struct{}, 2)
	eb.Subscribe("t6", func(ctx context.Context, e Event) error {
		calls <- struct{}{}
		if len(calls) == 1 {
			panic("boom")
		}
		return nil
	})
	eb.Publish(context.Background(), newTestEvent("t6"))
	eb.Publish(context.Background(), newTestEvent("t6"))
	if !waitTimeout(calls, 2*time.Second) {
		t.Fatal("dispatcher did not survive panic and process second event")
	}
	eb.Stop()
}
