// Package middleware — CORS handling.
//
// There is intentionally no wildcard ("*") path in production builds: a
// wildcard Access-Control-Allow-Origin combined with Allow-Credentials: true
// is either rejected by browsers or dangerous. Callers MUST provide an
// explicit allowlist via CORSMiddleware(origins) built from configuration.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// allowedHeaders lists request headers allowed across origins.
const allowedHeaders = "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With"

// allowedMethods lists HTTP methods allowed across origins.
const allowedMethods = "POST, OPTIONS, GET, PUT, DELETE, PATCH"

// CORSMiddleware returns a gin middleware that enforces the given origin
// allowlist. When the request Origin matches an entry, it is reflected back
// verbatim with credentials enabled. When it does not match, CORS headers
// are simply omitted (the browser will block the response).
//
// Passing an empty slice returns a middleware that rejects all cross-origin
// requests — safe-by-default behaviour when the operator forgot to set
// CORS_ALLOWED_ORIGINS.
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	originSet := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o != "" {
			originSet[o] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			if _, ok := originSet[origin]; ok {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Vary", "Origin")
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
				c.Writer.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
				c.Writer.Header().Set("Access-Control-Allow-Methods", allowedMethods)
			}
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// CORSMiddlewareWithOrigins is retained for backward compatibility and
// delegates to CORSMiddleware.
func CORSMiddlewareWithOrigins(allowedOrigins []string) gin.HandlerFunc {
	return CORSMiddleware(allowedOrigins)
}
