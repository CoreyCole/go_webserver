package handle

import (
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

func ServeBevy(c echo.Context) error {
	filename := c.Param("filename")
	if filename == "" {
		filename = "index.html"
	}
	fmt.Println("filename = " + filename)
	file, err := os.ReadFile("www/bevy/" + filename)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "File not found")
	}
	return c.HTMLBlob(http.StatusOK, file)
}
