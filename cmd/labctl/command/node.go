package command

import (
	"errors"

	"github.com/Netflix/p2plab/node"
	"github.com/urfave/cli"
)

var nodeCommand = cli.Command{
	Name:    "node",
	Aliases: []string{"n"},
	Usage:   "Manage nodes.",
	Subcommands: []cli.Command{
		{
			Name:   "label",
			Usage:  "Label nodes in a cluster for grouping in scenarios.",
			Action: labelNodesAction,
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name:  "add",
					Usage: "Adds a label to the matched nodes",
				},
				&cli.StringSliceFlag{
					Name:  "remove,rm",
					Usage: "Removes a label to the matched nodes",
				},
			},
		},
		{
			Name:   "ssh",
			Usage:  "SSH into a node.",
			Action: sshNodeAction,
		},
	},
}

func labelNodesAction(c *cli.Context) error {
	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)

	nset := node.NewSet()
	for i := 0; i < c.NArg(); i++ {
		id := c.Args().Get(i)

		n, err := cln.Node().Get(ctx, id)
		if err != nil {
			return err
		}
		nset.Add(n)
	}

	err = nset.Label(ctx, c.StringSlice("add"), c.StringSlice("remove"))
	if err != nil {
		return err
	}

	return nil
}

func sshNodeAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("node id must be provided")
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
