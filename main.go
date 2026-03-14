package main

import (
	"log"

	"github.com/kelseyhightower/envconfig"

	"github.com/coreycole/go_webserver/webserver"
)

type Config struct {
	Port          string `envconfig:"PORT"           default:"3001"`
	WebhookSecret string `envconfig:"WEBHOOK_SECRET"`
}

func main() {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal(err)
	}
	if err := webserver.Start(":"+cfg.Port, cfg.WebhookSecret); err != nil {
		log.Fatal(err)
	}
}
