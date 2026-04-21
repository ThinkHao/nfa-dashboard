package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/config"
)

type loginRateState struct {
	attempts    int
	firstSeenAt time.Time
	blockedTill time.Time
}

var (
	loginRateMu   sync.Mutex
	loginRateData = map[string]*loginRateState{}
)

// LoginRateLimit throttles brute-force attempts by client IP.
func LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.IsLoginRateLimitEnabled() {
			c.Next()
			return
		}

		now := time.Now()
		maxAttempts := config.GetLoginRateLimitMaxAttempts()
		window := time.Duration(config.GetLoginRateLimitWindowSecs()) * time.Second
		blockFor := time.Duration(config.GetLoginRateLimitBlockSecs()) * time.Second
		ip := c.ClientIP()

		loginRateMu.Lock()
		state, ok := loginRateData[ip]
		if !ok {
			state = &loginRateState{
				attempts:    0,
				firstSeenAt: now,
			}
			loginRateData[ip] = state
		}

		if now.After(state.firstSeenAt.Add(window)) {
			state.attempts = 0
			state.firstSeenAt = now
			state.blockedTill = time.Time{}
		}
		if now.Before(state.blockedTill) {
			retryAfter := int(state.blockedTill.Sub(now).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			loginRateMu.Unlock()
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"message": "too many login attempts, please retry later",
			})
			return
		}
		loginRateMu.Unlock()

		c.Next()

		if c.Writer.Status() != http.StatusUnauthorized {
			loginRateMu.Lock()
			if _, exists := loginRateData[ip]; exists {
				delete(loginRateData, ip)
			}
			loginRateMu.Unlock()
			return
		}

		loginRateMu.Lock()
		defer loginRateMu.Unlock()
		s, exists := loginRateData[ip]
		if !exists {
			s = &loginRateState{firstSeenAt: now}
			loginRateData[ip] = s
		}
		if now.After(s.firstSeenAt.Add(window)) {
			s.attempts = 0
			s.firstSeenAt = now
			s.blockedTill = time.Time{}
		}
		s.attempts++
		if s.attempts >= maxAttempts {
			s.blockedTill = now.Add(blockFor)
			s.attempts = 0
			s.firstSeenAt = now
		}
	}
}
