package MiddleWares

import (
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

		rl.mutex.Lock()
		defer rl.mutex.Unlock()

		record, exists := rl.records[clientIP]
		if !exists {
			record = &IPRequestRecord{
				Count:      0,
				LastAccess: time.Now(),
			}
			rl.records[clientIP] = record
		}

		if time.Since(record.LastAccess) > rl.window {
			record.Count = 0
			record.LastAccess = time.Now()
		}

		if record.Count >= rl.maxRequests {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests, please try again later",
			})
			c.Abort()
			return
		}

		record.Count++
		record.LastAccess = time.Now()
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
