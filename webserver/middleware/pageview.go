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
	// Only track known routes.
	if path == "/" || path == "/status" {
		return true
	}
	if strings.HasPrefix(path, "/md/") {
		return true
	}
	if strings.HasPrefix(path, "/games/") && strings.HasSuffix(path, "/game") {
		return true
	}
	return false
}

func hashIP(ip string) string {
	h := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(h[:8])
}
