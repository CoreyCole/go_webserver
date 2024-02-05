package handle

import (
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"

	lib "github.com/coreycole/go_webserver/webserver/lib"
	vi "github.com/coreycole/go_webserver/webserver/views"
)

func BevyLoadScript(js string, wasm string) templ.Component {
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

func GetGame(c echo.Context) error {
	filename := c.Param("filename")
	fmt.Println(">>>>>>>>>>> file = " + filename)
	js := fmt.Sprintf("/games/%s/%s.js", filename, filename)
	wasm := fmt.Sprintf("/games/%s/%s_bg.wasm", filename, filename)
	loadscript := BevyLoadScript(js, wasm)
	view := vi.BevyPage(js, wasm, loadscript)
	if err := view.Render(c.Request().Context(), c.Response().Writer); err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"Error rendering index: "+err.Error(),
		)
	}

	return nil
}
