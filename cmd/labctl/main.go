package main

import (
	"fmt"
	"os"

	"github.com/Netflix/p2plab/cmd/labctl/command"
)

func main() {
	app := command.App()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "labctl: %s\n", err)
		os.Exit(1)
	}
}
