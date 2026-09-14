package go26_http_server

import (
	"sync"
	"time"
)

// Limiter 是令牌桶限流器：按固定速率补充令牌，桶满即停。
type Limiter struct {
	mu     sync.Mutex
	now    func() time.Time
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

// NewLimiter 创建限流器：每秒补充 rate 个令牌，最多积攒 burst 个。
func NewLimiter(rate float64, burst int) *Limiter {
	return newLimiterWithClock(rate, burst, time.Now)
}

// newLimiterWithClock 允许注入时钟，测试和演示因此可以精确控制「过了多久」。
func newLimiterWithClock(rate float64, burst int, now func() time.Time) *Limiter {
	if rate <= 0 {
		rate = 1
	}
	if burst <= 0 {
		burst = 1
	}
	return &Limiter{
		now:    now,
		rate:   rate,
		burst:  float64(burst),
		tokens: float64(burst),
		last:   now(),
	}
}

// Allow 尝试取走一个令牌，取不到返回 false。
func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if elapsed := now.Sub(l.last).Seconds(); elapsed > 0 {
		l.tokens = min(l.tokens+elapsed*l.rate, l.burst)
		l.last = now
	}
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}
