package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	mu             sync.Mutex
	visitors       map[string]*visitor
	rate           rate.Limit
	burst          int
	trustedProxies []*net.IPNet
	stop           chan struct{}
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func NewRateLimiter(r rate.Limit, burst int, trustedProxies []string) (*RateLimiter, error) {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     r,
		burst:    burst,
		stop:     make(chan struct{}),
	}
	for _, raw := range trustedProxies {
		var ipnet *net.IPNet
		if strings.Contains(raw, "/") {
			_, parsed, err := net.ParseCIDR(raw)
			if err != nil {
				return nil, fmt.Errorf("invalid trusted proxy %q: %w", raw, err)
			}
			ipnet = parsed
		} else if ip := net.ParseIP(raw); ip != nil {
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			_, ipnet, _ = net.ParseCIDR(ip.String() + "/" + strconv.Itoa(bits))
		} else {
			return nil, fmt.Errorf("invalid trusted proxy %q", raw)
		}
		rl.trustedProxies = append(rl.trustedProxies, ipnet)
	}
	go rl.cleanup()
	return rl, nil
}

func (rl *RateLimiter) Close() { close(rl.stop) }

// SetLimit/SetBurst — for SIGHUP-reload. Applied to all visitors.
func (rl *RateLimiter) SetLimit(r rate.Limit) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.rate = r
	for _, v := range rl.visitors {
		v.limiter.SetLimit(r)
	}
}

func (rl *RateLimiter) SetBurst(b int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.burst = b
	for _, v := range rl.visitors {
		v.limiter.SetBurst(b)
	}
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-rl.stop:
			return
		case <-ticker.C:
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

func (rl *RateLimiter) isTrusted(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, n := range rl.trustedProxies {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// clientIP honors X-Forwarded-For only when RemoteAddr is a trusted proxy.
// Walks XFF right-to-left, returns the first non-trusted IP.
func (rl *RateLimiter) clientIP(r *http.Request) string {
	remote, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remote = r.RemoteAddr
	}
	if !rl.isTrusted(remote) {
		return remote
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(parts[i])
			if ip != "" && !rl.isTrusted(ip) {
				return ip
			}
		}
		return strings.TrimSpace(parts[0])
	}
	return remote
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := rl.clientIP(r)
		if !rl.get(ip).Allow() {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) get(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	v, ok := rl.visitors[ip]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(rl.rate, rl.burst)}
		rl.visitors[ip] = v
	}
	v.lastSeen = time.Now()
	return v.limiter
}
