package db

import (
	"expvar"
	"log"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/apache/iotdb-client-go/client"
)

var (
	sessionGetTimeoutTotal = expvar.NewInt("iotdb.session.get_timeout_total")
	sessionPoolRebuilds    = expvar.NewInt("iotdb.session.pool_rebuilds")
)

type SessionPool interface {
	GetSession() (client.Session, error)
	PutBack(session client.Session)
	Close()
}

type SessionPoolFactory func() SessionPool

const defaultRebuildThreshold = 5

const sessionPoolRetireDelay = 2 * time.Second

type SessionPoolWrapper struct {
	mu      sync.Mutex
	pool    SessionPool
	factory SessionPoolFactory

	threshold           int64
	consecutiveTimeouts atomic.Int64

	heldMu sync.Mutex
	held   []sessionRecord
}

type sessionRecord struct {
	session client.Session
	pool    SessionPool
}

func NewSessionPoolWrapper(factory SessionPoolFactory) *SessionPoolWrapper {
	return &SessionPoolWrapper{
		pool:      factory(),
		factory:   factory,
		threshold: defaultRebuildThreshold,
	}
}

func (w *SessionPoolWrapper) currentPool() SessionPool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.pool
}

func isSessionTimeout(err error) bool {
	return err != nil && strings.Contains(err.Error(), "timeout")
}

func (w *SessionPoolWrapper) GetSession() (client.Session, error) {
	pool := w.currentPool()
	session, err := pool.GetSession()
	if err != nil {
		if isSessionTimeout(err) {
			sessionGetTimeoutTotal.Add(1)
			if w.consecutiveTimeouts.Add(1) >= w.threshold {
				w.rebuildPool()
			}
		}
		return client.Session{}, err
	}
	w.consecutiveTimeouts.Store(0)
	w.heldMu.Lock()
	w.held = append(w.held, sessionRecord{session: session, pool: pool})
	w.heldMu.Unlock()
	return session, nil
}

func (w *SessionPoolWrapper) rebuildPool() {
	w.mu.Lock()
	if w.consecutiveTimeouts.Load() < w.threshold {
		w.mu.Unlock()
		return
	}
	old := w.pool
	w.pool = w.factory()
	w.consecutiveTimeouts.Store(0)
	w.mu.Unlock()

	sessionPoolRebuilds.Add(1)
	log.Printf("[SessionPoolWrapper] rebuilt IoTDB session pool after %d consecutive GetSession timeouts", w.threshold)

	if old == nil {
		return
	}
	go func() {
		time.Sleep(sessionPoolRetireDelay)
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[SessionPoolWrapper] recovered while closing retired pool: %v", r)
			}
		}()
		old.Close()
	}()
}

func (w *SessionPoolWrapper) PutBack(session client.Session) {
	var pool SessionPool
	w.heldMu.Lock()
	for idx, rec := range w.held {
		if reflect.DeepEqual(rec.session, session) {
			pool = rec.pool
			w.held = append(w.held[:idx], w.held[idx+1:]...)
			break
		}
	}
	w.heldMu.Unlock()
	if pool == nil {
		pool = w.currentPool()
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[SessionPoolWrapper] recovered while putting session back after pool rebuild: %v", r)
		}
	}()
	pool.PutBack(session)
}

func (w *SessionPoolWrapper) Close() {
	pool := w.currentPool()
	if pool != nil {
		pool.Close()
	}
}
