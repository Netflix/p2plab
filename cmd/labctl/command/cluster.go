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
	"github.com/rs/zerolog/log"
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
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "definition,d",
					Usage: "Create cluster from a cluster definition.",
				},
				&cli.IntFlag{
					Name:  "size,s",
					Usage: "Size of cluster.",
					Value: 3,
				},
				&cli.StringFlag{
					Name:  "instance-type,t",
					Usage: "EC2 Instance type of cluster.",
					Value: "t2.micro",
				},
				&cli.StringFlag{
					Name:  "region,r",
					Usage: "AWS Region to deploy to.",
					Value: "us-west-2",
				},
			},
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
			Name:    "update",
			Aliases: []string{"u"},
			Usage:   "Compiles a commit and updates a cluster to the new p2p app.",
			Action:  updateClusterAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "commit",
					Usage: "Specify commit to update to.",
					Value: "HEAD",
				},
			},
		},
	},
}

func createClusterAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("cluster id must be provided")
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	var options []p2plab.CreateClusterOption
	if c.IsSet("definition") {
		options = append(options,
			p2plab.WithClusterDefinition(c.String("definition")),
		)
	} else {
		options = append(options,
			p2plab.WithClusterSize(c.Int("size")),
			p2plab.WithClusterInstanceType(c.String("instance-type")),
			p2plab.WithClusterRegion(c.String("region")),
		)
	}

	cluster, err := control.Cluster().Create(CommandContext(c), c.Args().First(), options...)
	if err != nil {
		return err
	}

	log.Info().Msgf("Created cluster %q", cluster.Metadata().ID)
	return nil
}

func removeClusterAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("cluster id must be provided")
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	cluster, err := control.Cluster().Get(ctx, c.Args().First())
	if err != nil {
		return err
	}

	err = cluster.Remove(ctx)
	if err != nil {
		return err
	}

	log.Info().Msgf("Removed cluster %q", cluster.Metadata().ID)
	return nil
}

func listClusterAction(c *cli.Context) error {
	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	clusters, err := control.Cluster().List(CommandContext(c))
	if err != nil {
		return err
	}

	l := make([]interface{}, len(clusters))
	for i, c := range clusters {
		l[i] = c.Metadata()
	}

	return CommandPrinter(c).Print(l)
}

func queryClusterAction(c *cli.Context) error {
	if c.NArg() < 1 {
		return errors.New("cluster id must be provided")
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	cluster, err := control.Cluster().Get(ctx, c.Args().First())
	if err != nil {
		return err
	}

	rawQuery := "*"
	if c.NArg() == 2 {
		rawQuery = c.Args().Get(1)
	}

	q, err := query.Parse(rawQuery)
	if err != nil {
		return err
	}
	log.Info().Msgf("Parsed query as %q", q)

	var opts []p2plab.QueryOption
	addLabels := c.StringSlice("add")
	if len(addLabels) > 0 {
		opts = append(opts, p2plab.WithAddLabels(addLabels...))
	}

	removeLabels := c.StringSlice("remove")
	if len(removeLabels) > 0 {
		opts = append(opts, p2plab.WithRemoveLabels(removeLabels...))
	}

	nset, err := cluster.Query(ctx, q, opts...)
	if err != nil {
		return err
	}

	nodes := nset.Slice()
	l := make([]interface{}, len(nodes))
	for i, n := range nodes {
		l[i] = n.Metadata()
	}

	return CommandPrinter(c).Print(l)
}

func updateClusterAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("cluster id must be provided")
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	cluster, err := control.Cluster().Get(ctx, c.Args().First())
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

	log.Info().Msgf("Updated cluster %q", cluster.Metadata().ID)
	return nil
}
