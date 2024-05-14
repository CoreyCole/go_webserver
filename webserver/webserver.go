package webserver

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	h "github.com/coreycole/go_webserver/webserver/handle"
	m "github.com/coreycole/go_webserver/webserver/middleware"
)

func Start(port string) error {
	e := echo.New()

	// Middleware
	e.Use(middleware.Recover())
	e.Use(m.ZeroLog())

	e.GET("/", h.GetReact)
	e.Static("/", "public/")

	fmt.Println("starting on port", port)
	err := e.Start(port)
	e.Logger.Fatal(err)
	return err
}
