package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/coreycole/go_webserver/webserver/db"
	"github.com/coreycole/go_webserver/webserver/db/sqlc"
)

// PageViewMiddleware records page views for trackable routes.
func PageViewMiddleware(database *db.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			if shouldTrack(path) {
				visitorHash := hashIP(c.RealIP())
				_ = database.InsertPageView(
					c.Request().Context(),
					sqlc.InsertPageViewParams{
						Path:        path,
						VisitorHash: visitorHash,
					},
				)
			}
			return next(c)
		}
	}
}

func shouldTrack(path string) bool {
	// Skip static assets.
	for _, prefix := range []string{"/css/", "/js/", "/img/", "/favicon"} {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}
	// Skip SSE endpoints.
	if strings.HasSuffix(path, "/events") {
		return false
	}
	// Skip health checks.
	if path == "/health" {
		return false
	}
	// Skip files with extensions (static assets).
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		tail := path[idx:]
		if strings.Contains(tail, ".") {
			return false
		}
	}
	return true
}

func hashIP(ip string) string {
	h := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(h[:8])
}
