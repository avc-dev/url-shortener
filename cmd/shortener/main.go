package main

import (
	"fmt"
	"log"

	"github.com/avc-dev/url-shortener/internal/app"
)

var buildVersion string
var buildDate string
var buildCommit string

func main() {
	if buildVersion == "" {
		buildVersion = "N/A"
	}
	if buildDate == "" {
		buildDate = "N/A"
	}
	if buildCommit == "" {
		buildCommit = "N/A"
	}
	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", buildVersion, buildDate, buildCommit)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
