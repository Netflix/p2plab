package command

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/Netflix/p2plab/scenario"
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
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "name",
					Usage: "Name of the scenario, by default takes the name of the scenario definition.",
				},
			},
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
		return errors.New("scenario definition must be provided")
	}

	filename := c.Args().First()

	name := c.String("name")
	if name == "" {
		name = strings.TrimSuffix(filename, filepath.Ext(filename))
	}

	sdef, err := scenario.Parse(filename)
	if err != nil {
		return err
	}

	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	_, err = cln.Scenario().Create(ctx, name, sdef)
	if err != nil {
		return err
	}

	return nil
}

func removeScenarioAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("scenario name must be provided")
	}

	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	scenario, err := cln.Scenario().Get(ctx, c.Args().First())
	if err != nil {
		return err
	}

	err = scenario.Remove(ctx)
	if err != nil {
		return err
	}

	return nil
}

func listScenarioAction(c *cli.Context) error {
	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	scenarios, err := cln.Scenario().List(ctx)
	if err != nil {
		return err
	}

	return CommandPrinter(c).Print(scenarios)
}
