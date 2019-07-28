package command

import (
	"context"

	"github.com/Netflix/p2plab/labagent"
	"github.com/Netflix/p2plab/version"
	"github.com/urfave/cli"
)

func App() *cli.App {
	app := cli.NewApp()
	app.Name = "lab-agent"
	app.Version = version.Version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "log-level,l",
			Usage: "set the logging level [debug, info, warn, error, fatal, panic]",
		},
	}
	app.Action = agentAction
	return app
}

func agentAction(c *cli.Context) error {
	ctx := context.Background()

	agent, err := labagent.New()
	if err != nil {
		return err
	}

	return agent.Serve(ctx)
}
