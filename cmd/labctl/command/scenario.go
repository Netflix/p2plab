package command

import "github.com/urfave/cli"

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
	return nil
}

func removeScenarioAction(c *cli.Context) error {
	return nil
}

func listScenarioAction(c *cli.Context) error {
	return nil
}
