package command

import (
	"context"

	"github.com/Netflix/p2plab/labd"
	"github.com/Netflix/p2plab/version"
	"github.com/urfave/cli"
)

func App() *cli.App {
	app := cli.NewApp()
	app.Name = "labd"
	app.Version = version.Version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "log-level,l",
			Usage: "set the logging level [debug, info, warn, error, fatal, panic]",
		},
	}
	app.Action = daemonAction
	return app
}

func daemonAction(c *cli.Context) error {
	ctx := context.Background()

	daemon, err := labd.New()
	if err != nil {
		return err
	}

	return daemon.Serve(ctx)
}
