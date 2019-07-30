// Copyright 2019 Netflix, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"errors"

	"github.com/Netflix/p2plab"
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
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "commit",
					Usage: "Specify commit to update to.",
					Value: "HEAD",
				},
			},
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
	if c.NArg() < 1 {
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

	var q p2plab.Query
	if c.NArg() == 1 {
		q = query.All()
	} else {
		q, err = query.Parse(c.Args().Get(2))
		if err != nil {
			return err
		}
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

	var commit string
	if c.IsSet("commit") {
		commit = c.String("commit")
	}

	err = cluster.Update(ctx, commit)
	if err != nil {
		return err
	}

	return nil
}
