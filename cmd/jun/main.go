package main

import (
	"log"
	"os"

	"github.com/bencode/jun/internal/app"
	"github.com/bencode/jun/internal/tool"
)

func main() {
	// Before the app takes the single instance lock: a tool asks the Jun
	// that runs its terminal, which already holds it.
	if len(os.Args) > 1 && os.Args[1] == "tool" {
		os.Exit(tool.Main(os.Args[2:], os.Stdout, os.Stderr, os.Getenv))
	}
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
