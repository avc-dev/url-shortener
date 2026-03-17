package main

import (
	"log"
	"os"
)

func main() {
	// Uses log.Fatal and a helper — no direct os.Exit — no diagnostic expected.
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	return nil
}

// os.Exit inside a non-main function should not be flagged.
func shutdown() {
	os.Exit(0)
}
