package go27_http_client

import (
	"context"
	"math/rand/v2"
	"time"
)

// RetryPolicy 描述重试策略：最多试几次、退避多长、怎么等。
type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration

	// sleep 与 jitter 可注入：演示和测试不需要真的等待，结果也就能复现。
	sleep  func(context.Context, time.Duration) error
	jitter func(time.Duration) time.Duration
}

// NewRetryPolicy 返回默认策略：最多 4 次尝试，10ms 起步、指数退避、封顶 1s。
func NewRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 4,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    time.Second,
		sleep:       sleepContext,
		jitter:      randomJitter,
	}
}

// WithSleep 替换等待实现，用来跳过真实等待。
func (p RetryPolicy) WithSleep(sleep func(context.Context, time.Duration) error) RetryPolicy {
	p.sleep = sleep
	return p
}

// WithJitter 替换抖动函数；传 nil 表示不做抖动。
func (p RetryPolicy) WithJitter(jitter func(time.Duration) time.Duration) RetryPolicy {
	p.jitter = jitter
	return p
}

// Do 按策略执行 fn，返回实际尝试次数与最后一次错误。
//
// 只有 Retryable 判定为可重试的错误才会继续；上下文取消会立即停止。
func (p RetryPolicy) Do(ctx context.Context, fn func(context.Context) error) (int, error) {
	attempts := 0
	var lastErr error

	for attempt := 1; attempt <= p.MaxAttempts; attempt++ {
		attempts = attempt
		if lastErr = fn(ctx); lastErr == nil {
			return attempts, nil
		}
		if attempt == p.MaxAttempts || !Retryable(lastErr) {
			return attempts, lastErr
		}
		if err := p.sleep(ctx, p.delay(attempt, RetryAfter(lastErr))); err != nil {
			return attempts, err
		}
	}
	return attempts, lastErr
}

// delay 计算第 attempt 次失败后的等待时间。
//
// 优先尊重上游的 Retry-After；否则按 BaseDelay 指数增长，并加抖动——
// 抖动很关键：没有它，所有客户端会在同一时刻一起重试，把上游再打一遍。
func (p RetryPolicy) delay(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return min(retryAfter, p.MaxDelay)
	}

	delay := p.BaseDelay
	for i := 1; i < attempt; i++ {
		if delay >= p.MaxDelay/2 {
			delay = p.MaxDelay
			break
		}
		delay *= 2
	}
	if p.jitter != nil {
		delay = p.jitter(delay)
	}
	return delay
}

// sleepContext 等待 d，期间响应上下文取消。
func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// randomJitter 在 [d/2, d] 之间抖动。
func randomJitter(d time.Duration) time.Duration {
	if d <= 0 {
		return d
	}
	half := d / 2
	return half + time.Duration(rand.Int64N(int64(half)+1))
}
