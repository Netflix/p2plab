package command

import "github.com/urfave/cli"

var nodeCommand = cli.Command{
	Name:    "node",
	Aliases: []string{"n"},
	Usage:   "Manage nodes.",
	Subcommands: []cli.Command{
		{
			Name:    "label",
			Aliases: []string{"l"},
			Usage:   "Label nodes in a cluster for queries.",
			Action:  labelNodeAction,
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "List nodes in a cluster.",
			Action:  listNodeAction,
		},
		{
			Name:   "ssh",
			Usage:  "SSH into a node.",
			Action: sshNodeAction,
		},
	},
}

func labelNodeAction(c *cli.Context) error {
	return nil
}

func listNodeAction(c *cli.Context) error {
	return nil
}

func sshNodeAction(c *cli.Context) error {
	return nil
}
