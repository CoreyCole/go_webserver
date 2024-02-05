package handle

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v4"

	lib "github.com/coreycole/go_webserver/webserver/lib"
	vi "github.com/coreycole/go_webserver/webserver/views"
)

const (
	style     = "monokai"
	welcomeMd = "www/md/welcome.md"
)

func GetWelcome(c echo.Context) error {
	md, err := os.ReadFile(welcomeMd)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "File not found")
	}
	renderer, err := lib.NewMarkdownToHtmlRenderer(style)
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"Could not allocate markdown to HTML renderer",
		)
	}
	mdHTML := renderer.MarkdownBytesToHTML(md)
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"Error rendering markdown to html: "+err.Error(),
		)
	}

	// Use the Page templ component to construct the full page HTML
	mdComponent := lib.HTMLToComponent(mdHTML)
	view := vi.WelcomePage(mdComponent)

	if err := view.Render(c.Request().Context(), c.Response().Writer); err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"Error rendering index: "+err.Error(),
		)
	}

	return nil
}
