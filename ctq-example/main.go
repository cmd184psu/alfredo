package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/cmd184psu/alfredo/ctq"
)

func main() {
	name := filepath.Base(os.Args[0])

	switch name {
	case "ctqcli":
		ctq.RunCLI()
	case "ctq-coordinator":
		ctq.RunServices(true)
	case "ctq-worker":
		ctq.RunServices(false)
	default:
		log.Fatalf("unknown command name %q", name)
	}
}
