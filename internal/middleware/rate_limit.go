package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

type visitor struct {
	count    int
	lastSeen time.Time
}

type rateLimiter struct {
	visitors map[string]*visitor
	mu       sync.Mutex
	rate     int
	duration time.Duration
}

func RateLimit(rate int, duration time.Duration) fiber.Handler {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		duration: duration,
	}

	return func(c fiber.Ctx) error {
		if c.Path() == "/health" {
			return c.Next()
		}

		ip := c.IP()
		rl.mu.Lock()
		v, exists := rl.visitors[ip]
		if !exists {
			v = &visitor{count: 0, lastSeen: time.Now()}
			rl.visitors[ip] = v
		}

		if time.Since(v.lastSeen) > rl.duration {
			v.count = 0
			v.lastSeen = time.Now()
		}

		if v.count >= rl.rate {
			rl.mu.Unlock()
			return c.Status(429).JSON(fiber.Map{"error": "too many requests"})
		}

		v.count++
		v.lastSeen = time.Now()
		rl.mu.Unlock()

		return c.Next()
	}
}
