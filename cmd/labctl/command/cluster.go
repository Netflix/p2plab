package command

import (
	"errors"

	"github.com/Netflix/p2plab/query"
	"github.com/urfave/cli"
)

var clusterCommand = cli.Command{
	Name:    "cluster",
	Aliases: []string{"c"},
	Usage:   "Manage clusters.",
	Subcommands: []cli.Command{
		{
			Name:    "create",
			Aliases: []string{"c"},
			Usage:   "Creates a new cluster.",
			Action:  createClusterAction,
		},
		{
			Name:    "remove",
			Aliases: []string{"rm"},
			Usage:   "Remove clusters.",
			Action:  removeClusterAction,
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "List clusters.",
			Action:  listClusterAction,
		},
		{
			Name:    "query",
			Aliases: []string{"q"},
			Usage:   "Runs a query against a cluster and returns a set of matching nodes.",
			Action:  queryClusterAction,
		},
		{
			Name:    "update",
			Aliases: []string{"u"},
			Usage:   "Compiles a commit and updates a cluster to the new p2p app.",
			Action:  updateClusterAction,
		},
	},
}

func createClusterAction(c *cli.Context) error {
	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	_, err = cln.Cluster().Create(CommandContext(c))
	if err != nil {
		return err
	}

	return nil
}

func removeClusterAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("cluster id must be provided")
	}

	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	cluster, err := cln.Cluster().Get(ctx, c.Args().First())
	if err != nil {
		return err
	}

	err = cluster.Remove(ctx)
	if err != nil {
		return err
	}

	return nil
}

func listClusterAction(c *cli.Context) error {
	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	clusters, err := cln.Cluster().List(CommandContext(c))
	if err != nil {
		return err
	}

	return CommandPrinter(c).Print(clusters)
}

func queryClusterAction(c *cli.Context) error {
	if c.NArg() != 2 {
		return errors.New("cluster id and query must be provided")
	}

	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	cluster, err := cln.Cluster().Get(ctx, c.Args().First())
	if err != nil {
		return err
	}

	q, err := query.Parse(c.Args().Get(2))
	if err != nil {
		return err
	}

	nset, err := cluster.Query(ctx, q)
	if err != nil {
		return err
	}

	return CommandPrinter(c).Print(nset)
}

func updateClusterAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("cluster id must be provided")
	}

	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	cluster, err := cln.Cluster().Get(ctx, c.Args().First())
	if err != nil {
		return err
	}

	err = cluster.Update(ctx, c.Args().First())
	if err != nil {
		return err
	}

	return nil
}
