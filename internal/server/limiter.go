package server

import (
	"sync"
	"time"

	"github.com/gonejack/gshocksrv/internal/gshock"
)

type connectionLimiter struct {
	last     map[string]time.Time
	now      func() time.Time
	interval time.Duration

	mu sync.Mutex
}

func (t *connectionLimiter) allow(name string) bool {
	if !gshock.IsAlwaysConnected(name) {
		return true
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	if last, ok := t.last[name]; ok && now.Sub(last) <= t.interval {
		return false
	}
	t.last[name] = now
	return true
}

func newConnectionLimiter() *connectionLimiter {
	return &connectionLimiter{
		last:     make(map[string]time.Time),
		now:      time.Now,
		interval: 6 * time.Hour,
	}
}
