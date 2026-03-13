package handle

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"

	lib "github.com/coreycole/go_webserver/webserver/lib"
	vi "github.com/coreycole/go_webserver/webserver/view"
)

func GetMarkdownFile(c echo.Context) error {
	filename := c.Param("*")
	// Sanitize filename to prevent path traversal
	filename = filepath.Clean(filename)
	if strings.Contains(filename, "..") || filepath.IsAbs(filename) {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid filename")
	}
	md, err := os.ReadFile(filepath.Join("public/md", filename))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "File not found")
	}
	renderer, err := lib.NewMarkdownToHtmlRenderer()
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"Could not allocate markdown to HTML renderer",
		)
	}
	mdHTML := renderer.MarkdownBytesToHTML(md)

	// Use the Page templ component to construct the full page HTML
	mdComponent := lib.HTMLToComponent(mdHTML)
	view := vi.MarkdownPage(mdComponent)

	if err := view.Render(c.Request().Context(), c.Response().Writer); err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"Error rendering index: "+err.Error(),
		)
	}

	return nil
}
