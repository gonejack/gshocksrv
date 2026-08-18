package server

import (
	"testing"
	"time"
)

func TestConnectionLimiter(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	limiter := newConnectionLimiter()
	limiter.now = func() time.Time { return now }

	if !limiter.allow("CASIO ECB-30") {
		t.Fatal("first connection should be allowed")
	}
	if limiter.allow("CASIO ECB-30") {
		t.Fatal("second connection inside six hours should be denied")
	}
	now = now.Add(6*time.Hour + time.Second)
	if !limiter.allow("CASIO ECB-30") {
		t.Fatal("connection after six hours should be allowed")
	}
	if !limiter.allow("CASIO GW-B5600") || !limiter.allow("CASIO GW-B5600") {
		t.Fatal("ordinary watches should not be limited")
	}
}
