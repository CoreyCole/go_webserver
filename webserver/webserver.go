package webserver

import (
	"fmt"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"github.com/coreycole/go_md/webserver/handle"
)

func Start(port string) error {
	fmt.Println("starting on port", port)
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/health", handle.Health)
	e.GET("/md/:filename", handle.ServeMarkdown)

	// Start server
	e.Logger.Fatal(e.Start(port))
	return nil
}
