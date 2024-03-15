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

	// Routes
	e.GET("/", h.GetWelcome)
	e.GET("/health", h.GetHealth)
	// render markcown
	e.GET("/md/:filename", h.GetMarkdownFile)
	// game index pages e.g.
	// http://localhost:3000/games/giga_platformer/game
	e.GET("/games/:gameName/game", h.GetGame)

	// serve static files as a fallback (after all handlers)
	// game assets loaded with paths e.g.
	// http://localhost:3000/games/giga_platformer-97832db24b9e2bb6/assets/*
	e.Static("/", "public/")

	fmt.Println("starting on port", port)
	err := e.Start(port)
	e.Logger.Fatal(err)
	return err
}
