package service

import (
	"OMEGA3-IOT/internal/model"
	"sync"
	"time"
)

const (
	instanceAuthCacheTTL             = 30 * time.Second
	instanceAuthCacheJanitorInterval = 30 * time.Second
	instanceAuthCacheMaxEntries      = 10000
)

type instanceAuthCacheEntry struct {
	instance *model.Instance
	expireAt time.Time
}

// InstanceAuthCache caches MQTT auth lookups (UUID + VerifyHash) to skip a
// MySQL point query per device message. Every stored entry is a private
// snapshot: both Put and Get go through cloneInstanceForCache, because
// Properties.Items holds *TypedInstancePropertyItem pointers that
// updateDeviceProperties mutates in place (item.Value = ...). Returning or
// storing shared pointers would be a data race across MQTT workers.
type InstanceAuthCache struct {
	mu      sync.Mutex
	entries map[string]instanceAuthCacheEntry
}

func NewInstanceAuthCache() *InstanceAuthCache {
	c := &InstanceAuthCache{entries: make(map[string]instanceAuthCacheEntry)}
	go c.cleanupLoop()
	return c
}

// cleanupLoop evicts expired entries; mirrors TokenBlacklistService cleanup.
func (c *InstanceAuthCache) cleanupLoop() {
	ticker := time.NewTicker(instanceAuthCacheJanitorInterval)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for k, e := range c.entries {
			if now.After(e.expireAt) {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}

// Get returns a copy-on-access snapshot; callers may mutate the returned
// instance freely (including Properties.Items pointers) without affecting
// other workers or the cache itself.
func (c *InstanceAuthCache) Get(instanceUUID string) (*model.Instance, bool) {
	c.mu.Lock()
	e, ok := c.entries[instanceUUID]
	c.mu.Unlock()
	if !ok || time.Now().After(e.expireAt) {
		return nil, false
	}
	return cloneInstanceForCache(e.instance), true
}

// Put stores a clone of inst, so later in-place mutations of the caller's
// instance (e.g. updateDeviceProperties writing item.Value) never race with
// janitor scans or concurrent Get/Put readers.
func (c *InstanceAuthCache) Put(inst *model.Instance) {
	if inst == nil || inst.InstanceUUID == "" {
		return
	}
	c.mu.Lock()
	if len(c.entries) >= instanceAuthCacheMaxEntries {
		// Simple overflow guard: full reset; entries re-populate on demand.
		c.entries = make(map[string]instanceAuthCacheEntry)
	}
	c.entries[inst.InstanceUUID] = instanceAuthCacheEntry{
		instance: cloneInstanceForCache(inst),
		expireAt: time.Now().Add(instanceAuthCacheTTL),
	}
	c.mu.Unlock()
}

func (c *InstanceAuthCache) Invalidate(instanceUUID string) {
	c.mu.Lock()
	delete(c.entries, instanceUUID)
	c.mu.Unlock()
}

// cloneInstanceForCache returns a shallow struct copy with a cloned
// Properties.Items map and per-entry value copies of each item.
//
// Pointer/reference field audit of model.Instance:
//   - Properties.Items map[string]*TypedInstancePropertyItem — mutated in place
//     by updateDeviceProperties; map AND item pointers are re-created here.
//   - TypedInstancePropertyItem.Meta (PropertyMeta) — Range []float64, Enum
//     []string, CompiledPattern *regexp.Regexp, EnumSet map are shared
//     read-only: they are populated once at device-type load and never
//     mutated afterwards (only item.Value is replaced), so sharing is safe.
//   - TypedValue.V interface{} — only ever replaced wholesale (never mutated
//     in place by any writer), so sharing the old value across snapshots is
//     race-free.
//   - CreatedAt / UpdatedAt time.Time — value types, immutable.
//   - All other fields are scalars/strings copied by value.
func cloneInstanceForCache(inst *model.Instance) *model.Instance {
	if inst == nil {
		return nil
	}
	cp := *inst
	cp.Properties.Items = make(map[string]*model.TypedInstancePropertyItem, len(inst.Properties.Items))
	for k, item := range inst.Properties.Items {
		if item == nil {
			continue
		}
		itemCopy := *item
		cp.Properties.Items[k] = &itemCopy
	}
	return &cp
}
