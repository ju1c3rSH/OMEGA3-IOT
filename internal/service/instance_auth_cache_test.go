package service

import (
	"OMEGA3-IOT/internal/model"
	"sync"
	"testing"
	"time"
)

func newTestAuthInstance() *model.Instance {
	return &model.Instance{
		InstanceUUID: "11111111-2222-3333-4444-555555555555",
		VerifyHash:   "hash-a",
		Type:         "test-type",
		Properties: model.Properties{Items: map[string]*model.TypedInstancePropertyItem{
			"temperature": {
				Value: model.TypedValue{V: int64(1), Type: "int"},
				Meta:  model.PropertyMeta{Format: "int"},
			},
		}},
	}
}

func TestInstanceAuthCache(t *testing.T) {
	cache := NewInstanceAuthCache()
	inst := newTestAuthInstance()
	cache.Put(inst)

	if got, ok := cache.Get("11111111-2222-3333-4444-555555555555"); !ok || got.VerifyHash != "hash-a" {
		t.Fatalf("expected hit with VerifyHash=hash-a, got ok=%v hash=%v", ok, got.VerifyHash)
	}
	if _, ok := cache.Get("unknown"); ok {
		t.Fatal("expected miss for unknown uuid")
	}

	// Clone independence: mutating a returned copy must not leak into cache.
	got, _ := cache.Get("11111111-2222-3333-4444-555555555555")
	got.Properties.Items["temperature"].Value.V = int64(999)
	again, _ := cache.Get("11111111-2222-3333-4444-555555555555")
	if again.Properties.Items["temperature"].Value.V.(int64) == 999 {
		t.Fatal("cache leaked mutation from a returned copy (shared pointer)")
	}

	// Expiry: force expireAt into the past.
	cache.Put(inst)
	cache.mu.Lock()
	e := cache.entries["11111111-2222-3333-4444-555555555555"]
	e.expireAt = time.Now().Add(-time.Second)
	cache.entries["11111111-2222-3333-4444-555555555555"] = e
	cache.mu.Unlock()
	if _, ok := cache.Get("11111111-2222-3333-4444-555555555555"); ok {
		t.Fatal("expected miss after expiry")
	}

	// Concurrent Get + Put + Invalidate + mutation of returned copies.
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				if g, ok := cache.Get("11111111-2222-3333-4444-555555555555"); ok {
					if item := g.Properties.Items["temperature"]; item != nil {
						item.Value.V = int64(j + worker)
					}
				}
				cache.Put(inst)
				cache.Invalidate("11111111-2222-3333-4444-555555555555")
				cache.Invalidate("ghost")
			}
		}(i)
	}
	wg.Wait()

	cache.Put(inst)
	if got, ok := cache.Get("11111111-2222-3333-4444-555555555555"); !ok || got.InstanceUUID != inst.InstanceUUID {
		t.Fatal("expected hit after final Put")
	}
}
