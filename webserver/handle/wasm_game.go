package handle

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"

	lib "github.com/coreycole/go_webserver/webserver/lib"
	vi "github.com/coreycole/go_webserver/webserver/view"
)

const s3BaseURL = "https://coreycole-games.s3.us-west-2.amazonaws.com/games"

// maps short game names to their full directory names on S3
var gameDirs = map[string]string{
	"giga_platformer": "giga_platformer-7143ed686304a07e",
	"nessyclothes":    "nessyclothes-3d0f9d8535e29267",
}

func GetGame(c echo.Context) error {
	gameName := c.Param("gameName")
	gameDir, ok := gameDirs[gameName]
	if !ok {
		// check if it's already a full dir name
		for _, dir := range gameDirs {
			if dir != gameName {
				continue
			}
			gameDir = dir
			ok = true
			break
		}
	}
	if !ok {
		// try prefix match
		for name, dir := range gameDirs {
			if !strings.HasPrefix(name, gameName) {
				continue
			}
			gameDir = dir
			ok = true
			break
		}
	}
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "Game not found")
	}
	return serveGame(c, gameDir)
}

func serveGame(c echo.Context, gameDir string) error {
	js := fmt.Sprintf("%s/%s/%s.js", s3BaseURL, gameDir, gameDir)
	wasm := fmt.Sprintf("%s/%s/%s_bg.wasm", s3BaseURL, gameDir, gameDir)
	loadscript := bevyLoadScript(js, wasm)
	view := vi.BevyPage(
		js,
		wasm,
		loadscript,
	)
	if err := view.Render(c.Request().Context(), c.Response().Writer); err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"Error rendering page: "+err.Error(),
		)
	}

	return nil
}

func bevyLoadScript(js, wasm string) templ.Component {
	jsString := fmt.Sprintf(`<script type="module">import init from "%s";
init("%s").catch((error) => {
if (!error.message.startsWith("Using exceptions for control flow,")) {
  throw error;
}
});</script>`,
		js,
		wasm,
	)
	return lib.HTMLToComponent(jsString)
}
