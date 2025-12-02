package main

import (
	"log"

	"github.com/avc-dev/url-shortener/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
