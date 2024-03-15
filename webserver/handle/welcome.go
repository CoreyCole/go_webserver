package handle

import (
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"

	"github.com/coreycole/go_webserver/webserver/lib"
	vi "github.com/coreycole/go_webserver/webserver/view"
)

const (
	style     = "monokai"
	welcomeMd = "public/md/welcome.md"
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
	resumeHTML, err := lib.ResumeJSONToHTML("public/resume.json")
	if err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"Error rendering resume to html: "+err.Error(),
		)
	}
	welcomeContent := lib.HTMLToComponent(fmt.Sprintf("%s\n%s", mdHTML, resumeHTML))
	view := vi.WelcomePage(welcomeContent)

	if err := view.Render(c.Request().Context(), c.Response().Writer); err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"Error rendering index: "+err.Error(),
		)
	}

	return nil
}
