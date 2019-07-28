package command

import (
	"github.com/Netflix/p2plab/version"
	"github.com/urfave/cli"
)

func App() *cli.App {
	app := cli.NewApp()
	app.Name = "labctl"
	app.Version = version.Version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "address,a",
			Usage: "address for labd",
		},
		cli.StringFlag{
			Name:  "log-level,l",
			Usage: "set the logging level [debug, info, warn, error, fatal, panic]",
		},
		cli.StringFlag{
			Name:  "output,o",
			Usage: "set the output printer [table, json]",
			Value: "table",
		},
	}
	app.Commands = []cli.Command{
		clusterCommand,
		nodeCommand,
		scenarioCommand,
		benchmarkCommand,
	}

	// Setup tracers and context.
	AttachAppContext(app)

	// Setup output printer.
	AttachAppPrinter(app)

	return app
}
