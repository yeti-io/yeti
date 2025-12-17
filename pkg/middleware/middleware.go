package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func LoggerMiddleware(logger interface {
	Infow(msg string, keysAndValues ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if raw != "" {
			path = path + "?" + raw
		}

		logFields := []interface{}{
			"status", statusCode,
			"latency", latency,
			"client_ip", clientIP,
			"method", method,
			"path", path,
		}

		if errorMessage != "" {
			logFields = append(logFields, "error", errorMessage)
		}

		if statusCode >= 500 {
			logger.Errorw("HTTP Request", logFields...)
		} else {
			logger.Infow("HTTP Request", logFields...)
		}
	}
}

func RecoveryMiddleware(logger interface {
	Errorw(msg string, keysAndValues ...interface{})
}) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.Errorw("Panic recovered",
			"error", recovered,
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
		)
		c.AbortWithStatusJSON(500, gin.H{
			"error":      "internal server error",
			"error_code": "INTERNAL_ERROR",
		})
	})
}

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func generateRequestID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}
