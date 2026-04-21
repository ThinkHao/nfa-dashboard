package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/config"
)

// CORS 处理跨域请求的中间件
func CORS() gin.HandlerFunc {
	allowedOrigins := make(map[string]struct{})
	for _, origin := range config.GetCORSAllowedOrigins() {
		o := strings.TrimSpace(origin)
		if o == "" {
			continue
		}
		allowedOrigins[o] = struct{}{}
	}
	allowCredentials := config.GetCORSAllowCredentials()

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin != "" {
			if _, ok := allowedOrigins[origin]; ok {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Vary", "Origin")
				if allowCredentials {
					c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
				}
				c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With")
				c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
			} else if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
