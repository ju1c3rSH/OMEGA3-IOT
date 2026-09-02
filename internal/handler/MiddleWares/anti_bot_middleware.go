package MiddleWares

import (
	"OMEGA3-IOT/internal/types"
	"github.com/gin-gonic/gin"
	"net/http"
	"sync"
	"time"
)

type IPRequestRecord struct {
	Count      int
	LastAccess time.Time
}

type RateLimiter struct {
	records     map[string]*IPRequestRecord
	mutex       sync.RWMutex
	maxRequests int
	window      time.Duration
}

// NewRateLimiter creates a rate limiter allowing maxRequests per window per IP.
// window must be a time.Duration (e.g. 60*time.Second); values < time.Second are clamped to 1s
// to avoid accidental nanosecond windows (e.g. NewRateLimiter(15, 60) == 60ns) and ticker spin.
func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	if window < time.Second {
		window = time.Second
	}
	limiter := &RateLimiter{
		records:     make(map[string]*IPRequestRecord),
		maxRequests: maxRequests,
		window:      window,
	}

	//启动清理goroutine
	go limiter.cleanup()

	return limiter
}
func (rl *RateLimiter) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		rl.mutex.Lock()
		record, exists := rl.records[clientIP]
		if !exists {
			record = &IPRequestRecord{
				Count:      0,
				LastAccess: now,
			}
			rl.records[clientIP] = record
		}

		if now.Sub(record.LastAccess) > rl.window {
			record.Count = 0
			record.LastAccess = now
		}

		if record.Count >= rl.maxRequests {
			rl.mutex.Unlock()
			c.JSON(http.StatusTooManyRequests, types.NewErrorResponse(http.StatusTooManyRequests, "Too many requests, please try again later"))
			c.Abort()
			return
		}

		record.Count++
		record.LastAccess = now
		rl.mutex.Unlock()

		c.Next()
	}
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		// Collect expired keys under RLock to avoid holding exclusive lock during full map scan.
		rl.mutex.RLock()
		var expired []string
		for ip, record := range rl.records {
			if now.Sub(record.LastAccess) > rl.window*2 {
				expired = append(expired, ip)
			}
		}
		rl.mutex.RUnlock()
		if len(expired) == 0 {
			continue
		}
		// Batch delete under exclusive Lock; re-check condition to handle races.
		rl.mutex.Lock()
		for _, ip := range expired {
			if rec, ok := rl.records[ip]; ok && now.Sub(rec.LastAccess) > rl.window*2 {
				delete(rl.records, ip)
			}
		}
		rl.mutex.Unlock()
	}
}
