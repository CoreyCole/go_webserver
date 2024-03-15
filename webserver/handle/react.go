package handle

import (
	"net/http"

	"github.com/labstack/echo/v4"

	vi "github.com/coreycole/go_webserver/webserver/view"
)

func GetReact(c echo.Context) error {
	view := vi.ReactPage()
	if err := view.Render(c.Request().Context(), c.Response().Writer); err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
		)
	}
	return nil
}
