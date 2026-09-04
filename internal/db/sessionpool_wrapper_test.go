package db

import (
	"errors"
	"sync"
	"testing"

	"github.com/apache/iotdb-client-go/client"
)

var errGetTimeout = errors.New("get session timeout")

type scriptedPool struct {
	mu          sync.Mutex
	script      []error
	idx         int
	putBacks    []client.Session
	closeCalled bool
}

func (f *scriptedPool) GetSession() (client.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.idx >= len(f.script) {
		return client.Session{}, errors.New("script exhausted")
	}
	err := f.script[f.idx]
	f.idx++
	if err != nil {
		return client.Session{}, err
	}
	return client.Session{}, nil
}

func (f *scriptedPool) PutBack(session client.Session) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.putBacks = append(f.putBacks, session)
}

func (f *scriptedPool) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closeCalled = true
}

func (f *scriptedPool) putBackCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.putBacks)
}

type scriptFactory struct {
	mu    sync.Mutex
	pools []*scriptedPool
	first []error
	rest  []error
}

func newScriptFactory(first, rest []error) *scriptFactory {
	return &scriptFactory{first: first, rest: rest}
}

func (sf *scriptFactory) create() SessionPool {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	script := sf.rest
	if len(sf.pools) == 0 {
		script = sf.first
	}
	p := &scriptedPool{script: script}
	sf.pools = append(sf.pools, p)
	return p
}

func (sf *scriptFactory) count() int {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	return len(sf.pools)
}

func (sf *scriptFactory) firstPool() *scriptedPool {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	return sf.pools[0]
}

type alwaysTimeoutPool struct {
	mu       sync.Mutex
	getCalls int
}

func (f *alwaysTimeoutPool) GetSession() (client.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getCalls++
	return client.Session{}, errGetTimeout
}

func (f *alwaysTimeoutPool) PutBack(session client.Session) {}

func (f *alwaysTimeoutPool) Close() {}

type poolCounter struct {
	mu    sync.Mutex
	pools []*alwaysTimeoutPool
}

func (c *poolCounter) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.pools)
}

func TestSessionPoolWrapperRebuildsAfterConsecutiveTimeouts(t *testing.T) {
	c := &poolCounter{}
	w := NewSessionPoolWrapper(func() SessionPool {
		p := &alwaysTimeoutPool{}
		c.mu.Lock()
		c.pools = append(c.pools, p)
		c.mu.Unlock()
		return p
	})
	for i := 0; i < defaultRebuildThreshold; i++ {
		if _, err := w.GetSession(); !errors.Is(err, errGetTimeout) {
			t.Fatalf("call %d: expected timeout error, got %v", i+1, err)
		}
	}
	if got := c.count(); got != 2 {
		t.Fatalf("expected factory called 2 times (initial + 1 rebuild), got %d", got)
	}
	if w.consecutiveTimeouts.Load() != 0 {
		t.Fatalf("expected consecutive timeout counter reset after rebuild, got %d", w.consecutiveTimeouts.Load())
	}
}

func TestSessionPoolWrapperResetsCounterOnSuccess(t *testing.T) {
	sf := newScriptFactory([]error{
		errGetTimeout, errGetTimeout, errGetTimeout, errGetTimeout,
		nil,
		errGetTimeout, errGetTimeout, errGetTimeout, errGetTimeout, errGetTimeout,
	}, []error{nil})
	w := NewSessionPoolWrapper(sf.create)
	for i := 0; i < 4; i++ {
		if _, err := w.GetSession(); err == nil {
			t.Fatalf("call %d: expected timeout error", i+1)
		}
	}
	if got := sf.count(); got != 1 {
		t.Fatalf("4 consecutive timeouts must not rebuild, factory called %d times", got)
	}
	if _, err := w.GetSession(); err != nil {
		t.Fatalf("expected success to reset counter, got %v", err)
	}
	if w.consecutiveTimeouts.Load() != 0 {
		t.Fatalf("expected counter reset to 0, got %d", w.consecutiveTimeouts.Load())
	}
	for i := 0; i < defaultRebuildThreshold; i++ {
		if _, err := w.GetSession(); err == nil {
			t.Fatalf("timeout round call %d: expected timeout error", i+1)
		}
	}
	if got := sf.count(); got != 2 {
		t.Fatalf("expected rebuild after 5 more consecutive timeouts, factory called %d times", got)
	}
	if _, err := w.GetSession(); err != nil {
		t.Fatalf("expected success from rebuilt pool, got %v", err)
	}
	if got := sf.count(); got != 2 {
		t.Fatalf("expected no extra rebuild, factory called %d times", got)
	}
}

func TestSessionPoolWrapperPutBackRoutesToSourcePool(t *testing.T) {
	sf := newScriptFactory([]error{
		nil,
		errGetTimeout, errGetTimeout,
		nil,
	}, []error{nil})
	w := NewSessionPoolWrapper(sf.create)
	w.threshold = 2

	s, err := w.GetSession()
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if _, err := w.GetSession(); err == nil {
		t.Fatal("expected timeout error")
	}
	if _, err := w.GetSession(); err == nil {
		t.Fatal("expected timeout error")
	}
	if got := sf.count(); got != 2 {
		t.Fatalf("expected rebuild after 2 consecutive timeouts, factory called %d times", got)
	}
	if _, err := w.GetSession(); err != nil {
		t.Fatalf("expected success from rebuilt pool, got %v", err)
	}
	w.PutBack(s)
	if got := sf.firstPool().putBackCount(); got != 1 {
		t.Fatalf("expected stale session routed to source pool, source pool putBacks=%d", got)
	}
	if got := w.currentPool().(*scriptedPool).putBackCount(); got != 0 {
		t.Fatalf("expected rebuilt pool untouched, putBacks=%d", got)
	}
}

func TestSessionPoolWrapperNonTimeoutErrorsDoNotRebuild(t *testing.T) {
	connErr := errors.New("connection refused")
	sf := newScriptFactory([]error{
		connErr, connErr, connErr,
		connErr, connErr, connErr,
	}, []error{nil})
	w := NewSessionPoolWrapper(sf.create)
	for i := 0; i < 6; i++ {
		if _, err := w.GetSession(); err == nil {
			t.Fatalf("call %d: expected error", i+1)
		}
	}
	if got := sf.count(); got != 1 {
		t.Fatalf("non-timeout errors must not trigger rebuild, factory called %d times", got)
	}
	if w.consecutiveTimeouts.Load() != 0 {
		t.Fatalf("non-timeout errors must not increment timeout counter, got %d", w.consecutiveTimeouts.Load())
	}
}
