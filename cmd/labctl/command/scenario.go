package command

import (
	"errors"

	"github.com/urfave/cli"
)

var scenarioCommand = cli.Command{
	Name:    "scenario",
	Aliases: []string{"s"},
	Usage:   "Manage scenarios.",
	Subcommands: []cli.Command{
		{
			Name:    "create",
			Aliases: []string{"c"},
			Usage:   "Creates a new scenario.",
			Action:  createScenarioAction,
		},
		{
			Name:    "remove",
			Aliases: []string{"rm"},
			Usage:   "Remove scenarios.",
			Action:  removeScenarioAction,
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "List scenarios.",
			Action:  listScenarioAction,
		},
	},
}

func createScenarioAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("scenario name must be provided")
	}

	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	node, err := cln.Node().Get(ctx, c.Args().First())
	if err != nil {
		return err
	}

	err = node.SSH(ctx)
	if err != nil {
		return err
	}

	return nil
}

func removeScenarioAction(c *cli.Context) error {
	return nil
}

func listScenarioAction(c *cli.Context) error {
	return nil
}
