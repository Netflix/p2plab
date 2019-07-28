package main

import (
	"fmt"
	"os"

	"github.com/Netflix/p2plab/cmd/labd/command"
)

func main() {
	app := command.App()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "labd: %s\n", err)
		os.Exit(1)
	}
}
