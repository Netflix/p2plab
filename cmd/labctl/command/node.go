// Copyright 2019 Netflix, Inc.
//
// Licenodesed under the Apache Licenodese, Version 2.0 (the "Licenodese");
// you may not use this file except in compliance with the Licenodese.
// You may obtain a copy of the Licenodese at
//
//     http://www.apache.org/licenodeses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the Licenodese is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the Licenodese for the specific language governing permissionodes and
// limitationodes under the Licenodese.

package command

import (
	"encoding/json"
	"os"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/cliutil"
	"github.com/Netflix/p2plab/printer"
	"github.com/Netflix/p2plab/query"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var nodeCommand = cli.Command{
	Name:    "node",
	Aliases: []string{"n"},
	Usage:   "Manage nodes.",
	Subcommands: []cli.Command{
		{
			Name:      "inspect",
			Aliases:   []string{"i"},
			Usage:     "Displays detailed information on a node.",
			ArgsUsage: "<cluster> <id>",
			Action:    inspectNodeAction,
		},
		{
			Name:      "label",
			Aliases:   []string{"l"},
			Usage:     "Add or remove labels from nodes.",
			ArgsUsage: "<cluster>",
			Action:    labelNodesAction,
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "add",
					Usage: "Adds a label.",
				},
				cli.StringSliceFlag{
					Name:  "remove,rm",
					Usage: "Removes a label.",
				},
			},
		},
		{
			Name:      "list",
			Aliases:   []string{"ls"},
			Usage:     "List nodes.",
			ArgsUsage: "<cluster>",
			Action:    listNodeAction,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "query,q",
					Usage: "Runs a query to filter the listed nodes.",
				},
			},
		},
		{
			Name:      "update",
			Aliases:   []string{"u"},
			ArgsUsage: "<cluster>",
			Usage:     "Updates nodes with a peer definition",
			Action:    updateNodesAction,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "definition,d",
					Usage: "Update nodes from a peer definition.",
				},
				cli.StringFlag{
					Name:  "query,q",
					Usage: "Runs a query to update a subset of nodes.",
				},
				cli.StringFlag{
					Name:  "git-reference,ref",
					Usage: "Git reference of labapp.",
				},
				cli.StringSliceFlag{
					Name:  "transports,t",
					Usage: "Transports for libp2p [tcp, ws, quic]",
				},
				cli.StringSliceFlag{
					Name:  "muxers,m",
					Usage: "Muxers for libp2p [mplex, yamux]",
				},
				cli.StringSliceFlag{
					Name:  "security-transports,st",
					Usage: "Security transports for libp2p [tls, secio, noise]",
				},
				cli.StringFlag{
					Name:  "routing,r",
					Usage: "Routing for libp2p [nil, kaddht]",
				},
			},
		},
		{
			Name:      "ssh",
			Usage:     "SSH into a node.",
			ArgsUsage: "<cluster> <id>",
			Action:    sshNodeAction,
		},
	},
}

func inspectNodeAction(c *cli.Context) error {
	if c.NArg() != 2 {
		return errors.New("cluster and node id must be provided")
	}

	p, err := CommandPrinter(c, printer.OutputJSON)
	if err != nil {
		return err
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := cliutil.CommandContext(c)
	cluster := c.Args().First()
	id := c.Args().Get(1)
	node, err := control.Node().Get(ctx, cluster, id)
	if err != nil {
		return err
	}

	return p.Print(node.Metadata())
}

func labelNodesAction(c *cli.Context) error {
	if c.NArg() < 1 {
		return errors.New("cluster id must be provided")
	}

	p, err := CommandPrinter(c, printer.OutputTable)
	if err != nil {
		return err
	}

	var ids []string
	for i := 1; i < c.NArg(); i++ {
		ids = append(ids, c.Args().Get(i))
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := cliutil.CommandContext(c)
	cluster := c.Args().First()
	nodes, err := control.Node().Label(ctx, cluster, ids, c.StringSlice("add"), c.StringSlice("remove"))
	if err != nil {
		return err
	}

	l := make([]interface{}, len(nodes))
	for i, n := range nodes {
		l[i] = n.Metadata()
	}

	return p.Print(l)
}

func updateNodesAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("cluster id must be provided")
	}

	p, err := CommandPrinter(c, printer.OutputTable)
	if err != nil {
		return err
	}

	var opts []p2plab.ListOption
	ctx := cliutil.CommandContext(c)
	if c.IsSet("query") {
		q, err := query.Parse(ctx, c.String("query"))
		if err != nil {
			return err
		}

		opts = append(opts, p2plab.WithQuery(q.String()))
	}

	var pdef metadata.PeerDefinition
	if c.IsSet("definition") {
		f, err := os.Open(c.String("definition"))
		if err != nil {
			return err
		}
		defer f.Close()

		err = json.NewDecoder(f).Decode(&pdef)
		if err != nil {
			return err
		}
	}
	if c.IsSet("git-reference") {
		pdef.GitReference = c.String("git-reference")
	}
	if c.IsSet("transports") {
		pdef.Transports = c.StringSlice("transports")
	}
	if c.IsSet("muxers") {
		pdef.Muxers = c.StringSlice("muxers")
	}
	if c.IsSet("security-transports") {
		pdef.SecurityTransports = c.StringSlice("security-transports")
	}
	if c.IsSet("routing") {
		pdef.Routing = c.String("routing")
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	cid := c.Args().Get(0)
	cluster, err := control.Cluster().Get(ctx, cid)
	if err != nil {
		return err
	}

	nodes, err := cluster.Update(ctx, pdef, opts...)
	if err != nil {
		return err
	}

	l := make([]interface{}, len(nodes))
	for i, n := range nodes {
		l[i] = n.Metadata()
	}

	return p.Print(l)
}

func listNodeAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("cluster id must be provided")
	}

	p, err := CommandPrinter(c, printer.OutputTable)
	if err != nil {
		return err
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	var opts []p2plab.ListOption
	ctx := cliutil.CommandContext(c)
	if c.IsSet("query") {
		q, err := query.Parse(ctx, c.String("query"))
		if err != nil {
			return err
		}

		opts = append(opts, p2plab.WithQuery(q.String()))
	}

	cluster := c.Args().First()
	nodes, err := control.Node().List(ctx, cluster, opts...)
	if err != nil {
		return err
	}

	l := make([]interface{}, len(nodes))
	for i, n := range nodes {
		l[i] = n.Metadata()
	}

	return p.Print(l)
}

func sshNodeAction(c *cli.Context) error {
	if c.NArg() != 2 {
		return errors.New("cluster id and node id must be provided")
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := cliutil.CommandContext(c)
	node, err := control.Node().Get(ctx, c.Args().Get(0), c.Args().Get(1))
	if err != nil {
		return err
	}

	err = node.SSH(ctx)
	if err != nil {
		return err
	}

	return nil
}
