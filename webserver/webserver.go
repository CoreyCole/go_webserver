package webserver

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	ha "github.com/coreycole/go_webserver/webserver/handle"
	mi "github.com/coreycole/go_webserver/webserver/middleware"
)

func Start(port string) error {
	e := echo.New()

	// Middleware
	e.Use(middleware.Recover())
	e.Use(mi.ZeroLog())

	// Routes
	e.GET("/", ha.GetWelcome)
	e.GET("/health", ha.GetHealth)
	// render markcown
	e.GET("/md/:filename", ha.GetMarkdownFile)
	// game index pages e.g.
	// http://localhost:3000/games/giga_platformer/game
	e.GET("/games/:gameName/game", ha.GetGame)

	// serve static files as a fallback (after all handlers)
	// game assets loaded with paths e.g.
	// http://localhost:3000/games/giga_platformer-97832db24b9e2bb6/assets/*
	e.Static("/", "static/")

	fmt.Println("starting on port", port)
	err := e.Start(port)
	e.Logger.Fatal(err)
	return err
}
