package MiddleWares

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// SampledLogger 异步采样日志：health 按 IP 1/min，其它全量；缓冲 4096 满时丢弃并计数
type SampledLogger struct {
	ch        chan string
	dropped   atomic.Int64
	lastLog   sync.Map // ip -> time.Time
	bufWriter *bufio.Writer
	mu        sync.Mutex
}

const sampledLoggerBuffer = 4096
const healthSampleWindow = time.Minute

func NewSampledLogger() *SampledLogger {
	sl := &SampledLogger{
		ch:        make(chan string, sampledLoggerBuffer),
		bufWriter: bufio.NewWriterSize(os.Stderr, 32*1024),
	}
	go sl.run()
	go sl.cleanupLoop()
	// 定期 flush，避免缓冲丢失
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			sl.mu.Lock()
			_ = sl.bufWriter.Flush()
			sl.mu.Unlock()
			if d := sl.dropped.Load(); d > 0 {
				// 暴露丢弃计数：每 10s 打印一次，避免刷屏
				sl.logSync(fmt.Sprintf("[SampledLogger] dropped %d access logs (buffer full 4096)", d))
			}
		}
	}()
	return sl
}

func (sl *SampledLogger) logSync(s string) {
	sl.mu.Lock()
	_, _ = sl.bufWriter.WriteString(s + "\n")
	_ = sl.bufWriter.Flush()
	sl.mu.Unlock()
}

func (sl *SampledLogger) run() {
	for line := range sl.ch {
		sl.mu.Lock()
		_, _ = sl.bufWriter.WriteString(line + "\n")
		sl.mu.Unlock()
	}
}

func (sl *SampledLogger) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		sl.lastLog.Range(func(k, v interface{}) bool {
			if now.Sub(v.(time.Time)) > 2*healthSampleWindow {
				sl.lastLog.Delete(k)
			}
			return true
		})
	}
}

// Middleware gin 中间件：异步写，health 按 IP 采样
func (sl *SampledLogger) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		// health 采样：每 IP 每分钟最多一次
		if path == "/api/v1/health" || path == "/api/v1/test" {
			ip := c.ClientIP()
			now := time.Now()
			if v, ok := sl.lastLog.Load(ip); ok {
				if now.Sub(v.(time.Time)) < healthSampleWindow {
					return
				}
			}
			sl.lastLog.Store(ip, now)
		}

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		ip := c.ClientIP()
		line := fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %s",
			start.Format("2006/01/02 - 15:04:05"),
			status,
			latency,
			ip,
			method,
			path,
		)
		if len(c.Errors) > 0 {
			line += " | " + c.Errors.String()
		}

		select {
		case sl.ch <- line:
		default:
			sl.dropped.Add(1)
		}
	}
}

// DroppedCount 供 metrics / debug 查询
func (sl *SampledLogger) DroppedCount() int64 {
	return sl.dropped.Load()
}
