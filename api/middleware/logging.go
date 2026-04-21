package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger returns a gin middleware that emits one structured log line
// per request via the supplied slog.Logger. Requests to /health, /ready, and
// /live are intentionally skipped to keep probe traffic out of the log
// stream.
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}

	skip := map[string]struct{}{
		"/health": {},
		"/ready":  {},
		"/live":   {},
	}

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		c.Next()

		if _, ok := skip[path]; ok {
			return
		}

		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}

		attrs := []any{
			slog.Int("status", status),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("ip", c.ClientIP()),
			slog.Duration("duration", time.Since(start)),
		}
		if rawQuery != "" {
			attrs = append(attrs, slog.String("query", rawQuery))
		}
		if errs := c.Errors.ByType(gin.ErrorTypePrivate).String(); errs != "" {
			attrs = append(attrs, slog.String("errors", errs))
		}

		logger.LogAttrs(c.Request.Context(), level, "http_request", toAttrs(attrs)...)
	}
}

// toAttrs converts a variadic list of slog.Attr-shaped values to a []slog.Attr.
// slog.LogAttrs is faster than variadic Log because it skips reflection.
func toAttrs(vals []any) []slog.Attr {
	out := make([]slog.Attr, 0, len(vals))
	for _, v := range vals {
		if a, ok := v.(slog.Attr); ok {
			out = append(out, a)
		}
	}
	return out
}
