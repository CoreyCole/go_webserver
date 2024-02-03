package webserver

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	hd "github.com/coreycole/go_md/webserver/handle"
	mw "github.com/coreycole/go_md/webserver/middleware"
)

func Start(port string) error {
	fmt.Println("starting on port", port)
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Recover())
	e.Use(mw.ZeroLog())

	// Routes
	e.GET("/health", hd.Health)
	e.GET("/md/:filename", hd.ServeMarkdown)
	e.Static("/bevy", "www/bevy")

	// Start server
	e.Logger.Fatal(e.Start(port))
	return nil
}
