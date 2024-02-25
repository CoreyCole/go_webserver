package handle

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"

	lib "github.com/coreycole/go_webserver/webserver/lib"
	vi "github.com/coreycole/go_webserver/webserver/views"
)

func GetGame(c echo.Context) error {
	gameName := c.Param("gameName")
	baseDir := "static/games"
	exactMatch, err := os.Stat(filepath.Join(baseDir, gameName))
	if err == nil && exactMatch.IsDir() {
		fmt.Printf("exact match\n")
		return serveGame(c, gameName)
	}
	gameDir, err := findGameDir(baseDir, gameName)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Game not found")
	}
	// Redirect to the URL with the full game directory name
	redirectURL := fmt.Sprintf("/games/%s/game", gameDir)
	fmt.Printf("redirectURl = %s\n", redirectURL)
	return c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func serveGame(c echo.Context, gameDir string) error {
	js := fmt.Sprintf("/games/%s/%s.js", gameDir, gameDir)
	wasm := fmt.Sprintf("/games/%s/%s_bg.wasm", gameDir, gameDir)
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

func findGameDir(gamesDir string, gameName string) (string, error) {
	files, err := os.ReadDir(gamesDir)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if file.IsDir() && strings.HasPrefix(file.Name(), gameName) {
			return file.Name(), nil
		}
	}

	return "", fmt.Errorf("directory for game %s not found", gameName)
}

func bevyLoadScript(js string, wasm string) templ.Component {
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
