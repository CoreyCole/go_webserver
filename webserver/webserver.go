package webserver

import (
	"fmt"
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/coreycole/go_webserver/webserver/db"
	h "github.com/coreycole/go_webserver/webserver/handle"
	m "github.com/coreycole/go_webserver/webserver/middleware"
	"github.com/coreycole/go_webserver/webserver/webhook"
)

func Start(port, webhookSecret string) error {
	database, err := db.New("data/go_webserver.db")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	e := echo.New()

	// Middleware
	e.Use(middleware.Recover())
	e.Use(m.ZeroLog())
	e.Use(m.PageViewMiddleware(database))

	// Status handler (background snapshot writer starts on creation)
	status := h.NewStatusHandler(database)

	// Routes
	e.GET("/", h.GetWelcome)
	e.GET("/health", h.GetHealth)
	// render markdown (wildcard supports nested paths like /md/blog/post.md)
	e.GET("/md/*", h.GetMarkdownFile)
	// game pages — assets served from S3
	e.GET("/games/:gameName/game", h.GetGame)
	// status page with SSE metrics
	e.GET("/status", status.GetStatus)
	e.GET("/status/events", status.GetStatusEvents)
	e.POST("/status/graph", status.PostGraphUpdate)

	// webhook for auto-deploy
	wh := webhook.New(webhookSecret)
	e.POST("/webhook/github", wh.HandleGitHub)

	// serve static files as a fallback (after all handlers)
	e.Static("/", "public/")

	fmt.Println("starting on port", port)
	return e.Start(port)
}
