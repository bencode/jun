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
		os.Exit(tool.Main(os.Args[2:], os.Stdin, os.Stdout, os.Stderr, os.Getenv))
	}
	if wd, err := os.Getwd(); err == nil {
		if hint := tool.MistakenCommand(os.Args[1:], wd); hint != "" {
			os.Stderr.WriteString(hint)
			os.Exit(2)
		}
	}
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
