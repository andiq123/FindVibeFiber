package providers

import (
	"context"
	"sync"
	"time"
)

// pacer serializes bursts so parallel explore/radio resolves don't stampede a provider.
type pacer struct {
	mu     sync.Mutex
	last   time.Time
	minGap time.Duration
}

func newPacer(minGap time.Duration) *pacer {
	return &pacer{minGap: minGap}
}

func (p *pacer) wait(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.last.IsZero() {
		wait := p.minGap - time.Since(p.last)
		if wait > 0 {
			timer := time.NewTimer(wait)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	p.last = time.Now()
	return nil
}
