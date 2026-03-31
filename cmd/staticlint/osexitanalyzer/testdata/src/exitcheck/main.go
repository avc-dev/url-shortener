package main

import "os"

func main() {
	os.Exit(1) // want `direct os.Exit call in main function of main package is prohibited`
}

// helper calls os.Exit but is NOT the main function — should not be flagged.
func helper() {
	os.Exit(0)
}
