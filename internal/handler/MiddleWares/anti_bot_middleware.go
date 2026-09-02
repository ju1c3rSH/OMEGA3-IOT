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

func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
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
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for range ticker.C {
		rl.mutex.Lock()
		now := time.Now()
		for ip, record := range rl.records {
			if now.Sub(record.LastAccess) > rl.window*2 {
				delete(rl.records, ip)
			}
		}
		rl.mutex.Unlock()
	}
}
