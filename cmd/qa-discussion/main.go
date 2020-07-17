package main

import (
	"os"

	"github.com/clear-ness/qa-discussion/cmd/qa-discussion/commands"
)

func main() {
	if err := commands.Run(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
