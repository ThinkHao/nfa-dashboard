package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/internal/model"
)

// Audit records basic operation logs into operation_logs table
// It should be registered globally to log all requests. If user info exists in context, it will be associated.
func Audit() gin.HandlerFunc {
	// duplicate the context key to avoid import cycle
	const contextUserKey = "currentUser"
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		// After handlers
		latency := time.Since(start)
		status := c.Writer.Status()
		success := int8(1)
		if status >= 400 {
			success = 0
		}

		var userIDPtr *uint64
		if v, ok := c.Get(contextUserKey); ok && v != nil {
			if u, ok := v.(*model.User); ok && u != nil {
				userIDPtr = &u.ID
			}
		}

		method := c.Request.Method
		path := c.Request.URL.Path
		ip := c.ClientIP()
		ua := c.Request.UserAgent()

		latMS := int(latency / time.Millisecond)
		var errMsg *string
		if last := c.Errors.Last(); last != nil {
			s := last.Error()
			errMsg = &s
		}

		log := model.OperationLog{
			UserID:     userIDPtr,
			Method:     method,
			Path:       path,
			StatusCode: status,
			Success:    success,
			LatencyMS:  &latMS,
		}
		if ip != "" {
			log.IP = &ip
		}
		if ua != "" {
			log.UserAgent = &ua
		}
		if errMsg != nil {
			log.ErrorMessage = errMsg
		}

		// Best-effort insert; do not block request even if DB fails
		_ = model.DB.Create(&log).Error
	}
}
