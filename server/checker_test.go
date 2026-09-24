package server

import (
	"testing"
	"time"
)

func TestConnectionLimiter(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	checker := newConnectCheck()
	checker.now = func() time.Time { return now }

	if !checker.allow("CASIO ECB-30") {
		t.Fatal("first connection should be allowed")
	}
	if checker.allow("CASIO ECB-30") {
		t.Fatal("second connection inside six hours should be denied")
	}
	now = now.Add(6*time.Hour + time.Second)
	if !checker.allow("CASIO ECB-30") {
		t.Fatal("connection after six hours should be allowed")
	}
	if !checker.allow("CASIO GW-B5600") || !checker.allow("CASIO GW-B5600") {
		t.Fatal("ordinary watches should not be limited")
	}
}
