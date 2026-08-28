package main

import (
	"flag"
	"fmt"
	"os"

	"codea/tui/internal/update"
)

func main() {
	home := flag.String("home", "", "Codea home")
	target := flag.String("target", "", "installed version directory")
	flag.Parse()
	if *home == "" || *target == "" {
		fmt.Fprintln(os.Stderr, "-home and -target are required")
		os.Exit(2)
	}
	if err := update.NewPlatformSwitcher(*home).Switch(*target); err != nil {
		fmt.Fprintf(os.Stderr, "switch current version: %v\n", err)
		os.Exit(1)
	}
}
