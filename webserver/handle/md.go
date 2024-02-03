package handle

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v4"

	"github.com/coreycole/go_md/webserver/lib"
	"github.com/coreycole/go_md/webserver/views"
)

func ServeMarkdown(c echo.Context) error {
	filename := c.Param("filename")
	style := c.QueryParam("style")
	md, err := os.ReadFile("www/md/" + filename)
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
	view := views.MarkdownPage(mdComponent)

	if err := view.Render(c.Request().Context(), c.Response().Writer); err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"Error rendering index: "+err.Error(),
		)
	}

	return nil
}
