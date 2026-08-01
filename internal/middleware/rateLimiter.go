package middleware

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

//AI-Generated Review

// clientLimiter pairs a rate limiter with a last-seen timestamp
// so we can evict stale entries and avoid unbounded memory growth.
type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter manages per-IP rate limiters with automatic cleanup.
type IPRateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientLimiter
	rate    rate.Limit
	burst   int
	ttl     time.Duration
}

// NewIPRateLimiter creates a limiter and starts a background cleanup goroutine.
// r = sustained requests/second, burst = max burst size, ttl = idle eviction time.
func NewIPRateLimiter(r rate.Limit, burst int, ttl time.Duration) *IPRateLimiter {
	rl := &IPRateLimiter{
		clients: make(map[string]*clientLimiter),
		rate:    r,
		burst:   burst,
		ttl:     ttl,
	}
	go rl.cleanupLoop()
	return rl
}

// getLimiter retrieves or creates the rate limiter for a given IP.
func (rl *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cl, exists := rl.clients[ip]
	if !exists {
		cl = &clientLimiter{limiter: rate.NewLimiter(rl.rate, rl.burst)}
		rl.clients[ip] = cl
	}
	cl.lastSeen = time.Now()
	return cl.limiter
}

// cleanupLoop periodically evicts limiters for IPs that haven't been
// seen recently, preventing unbounded memory growth.
func (rl *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.ttl / 2)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, cl := range rl.clients {
			if time.Since(cl.lastSeen) > rl.ttl {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimiter returns a Gin middleware that enforces per-IP rate limiting.
func RateLimiter(rl *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := rl.getLimiter(ip)

		// Reserve() gives us the wait duration — essential for a correct
		// Retry-After header. Allow() would discard that information.
		reservation := limiter.Reserve()
		delay := reservation.Delay()

		if delay > 0 {
			// Cancel immediately: we're rejecting, not queuing.
			reservation.Cancel()

			retryAfter := int(math.Ceil(delay.Seconds()))
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate_limit_exceeded",
				"retry_after": retryAfter,
			})
			return
		}

		c.Next()
	}
}
