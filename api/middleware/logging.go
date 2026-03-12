// api/middleware/logging.go

package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		startTime := time.Now()

		// Get request details
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		method := c.Request.Method
		clientIP := c.ClientIP()

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(startTime)

		// Get status code
		statusCode := c.Writer.Status()

		// Get error if exists
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		// Build log message
		if raw != "" {
			path = path + "?" + raw
		}

		// Color-code based on status
		statusColor := getStatusColor(statusCode)

		// Log format: [timestamp] status method path latency ip error
		fmt.Printf("[%s] %s%d%s %s %s %v %s %s\n",
			time.Now().Format("2006-01-02 15:04:05"),
			statusColor,
			statusCode,
			resetColor(),
			method,
			path,
			latency,
			clientIP,
			errorMessage,
		)
	}
}

// StructuredLoggingMiddleware provides structured logging
func StructuredLoggingMiddleware(logger interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(startTime)

		// Get request details
		statusCode := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		// Log using structured logger if available
		// This is a simple implementation - can be enhanced with actual logger
		fmt.Printf("[GIN] %s | %3d | %13v | %15s | %-7s %s %s\n",
			time.Now().Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
			errorMessage,
		)
	}
}

// getStatusColor returns ANSI color code based on HTTP status
func getStatusColor(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "\033[32m" // Green
	case status >= 300 && status < 400:
		return "\033[36m" // Cyan
	case status >= 400 && status < 500:
		return "\033[33m" // Yellow
	case status >= 500:
		return "\033[31m" // Red
	default:
		return "\033[37m" // White
	}
}

// resetColor returns ANSI reset color code
func resetColor() string {
	return "\033[0m"
}
