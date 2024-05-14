package main

import (
	"github.com/coreycole/go_webserver/webserver"
)

func main() {
	err := webserver.Start(":3030")
	if err != nil {
		panic(err)
	}
}
