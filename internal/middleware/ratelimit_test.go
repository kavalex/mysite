package middleware

import (
	"testing"
	"testing/synctest"
	"time"

	"golang.org/x/time/rate"
)

func TestRateLimiterBurst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rl, err := NewRateLimiter(rate.Every(time.Second), 2, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer rl.Close()

		ip := "203.0.113.1"
		if !rl.get(ip).Allow() {
			t.Fatal("first request should be allowed")
		}
		if !rl.get(ip).Allow() {
			t.Fatal("second request should be allowed (burst=2)")
		}
		if rl.get(ip).Allow() {
			t.Fatal("third request should be rate-limited")
		}

		synctest.Sleep(time.Second) // fake clock — advances instantly
		if !rl.get(ip).Allow() {
			t.Fatal("after 1s a new token should be available")
		}
	})
}
