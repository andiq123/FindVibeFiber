package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

type rateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	rate     int           // requests per duration
	duration time.Duration
	cleanup  *time.Ticker
}

type visitor struct {
	count    int
	lastSeen time.Time
}

func NewRateLimiter(rate int, duration time.Duration) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		duration: duration,
		cleanup:  time.NewTicker(1 * time.Minute),
	}

	// Cleanup old visitors
	go func() {
		for range rl.cleanup.C {
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if time.Since(v.lastSeen) > 10*time.Minute {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()

	return rl
}

func (rl *rateLimiter) getVisitor(ip string) *visitor {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		v = &visitor{
			count:    0,
			lastSeen: time.Now(),
		}
		rl.visitors[ip] = v
	}
	return v
}

func (rl *rateLimiter) allow(ip string) bool {
	v := rl.getVisitor(ip)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Reset count if duration has passed
	if time.Since(v.lastSeen) > rl.duration {
		v.count = 0
		v.lastSeen = time.Now()
	}

	if v.count >= rl.rate {
		return false
	}

	v.count++
	v.lastSeen = time.Now()
	return true
}

func RateLimit(rate int, duration time.Duration) fiber.Handler {
	limiter := NewRateLimiter(rate, duration)

	return func(c fiber.Ctx) error {
		ip := c.IP()
		if !limiter.allow(ip) {
			return c.Status(429).JSON(fiber.Map{
				"error": "too many requests",
			})
		}
		return c.Next()
	}
}
