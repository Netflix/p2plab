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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/nodes"
	"github.com/Netflix/p2plab/pkg/cliutil"
	"github.com/Netflix/p2plab/query"
	"github.com/urfave/cli"
)

var debugCommand = cli.Command{
	Name:    "debug",
	Aliases: []string{"d"},
	Usage:   "Debugging tools.",
	Hidden:  true,
	Subcommands: []cli.Command{
		{
			Name:      "peer",
			Aliases:   []string{"p"},
			Usage:     "Retrieves the peer info from a labapp",
			ArgsUsage: " ",
			Action:    peerInfoAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "app-addr",
					Usage: "address for labapp's HTTP server",
					Value: "http://localhost:7003",
				},
			},
		},
		{
			Name:      "run",
			Aliases:   []string{"r"},
			Usage:     "Runs a task on a labapp.",
			ArgsUsage: "<task> <subject>",
			Action:    runTaskAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "app-addr",
					Usage: "address for labapp's HTTP server",
					Value: "http://localhost:7003",
				},
			},
		},
		{
			Name:      "connect",
			Aliases:   []string{"c"},
			Usage:     "Connects a cluster together",
			ArgsUsage: "<cluster>",
			Action:    connectAction,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "query,q",
					Usage: "Runs a query to filter the listed nodes.",
				},
			},
		},
	},
}

func peerInfoAction(c *cli.Context) error {
	app, err := ResolveApp(c, c.String("app-addr"))
	if err != nil {
		return err
	}

	ctx := cliutil.CommandContext(c)
	peerInfo, err := app.PeerInfo(ctx)
	if err != nil {
		return err
	}

	content, err := json.MarshalIndent(&peerInfo, "", "    ")
	if err != nil {
		return err
	}

	fmt.Printf("Peer info:\n%s\n", string(content))
	return nil
}

func runTaskAction(c *cli.Context) error {
	if c.NArg() != 2 {
		return errors.New("task type and subject must be provided")
	}

	app, err := ResolveApp(c, c.String("app-addr"))
	if err != nil {
		return err
	}

	ctx := cliutil.CommandContext(c)
	err = app.Run(ctx, metadata.Task{
		Type:    metadata.TaskType(c.Args().Get(0)),
		Subject: c.Args().Get(1),
	})
	if err != nil {
		return err
	}

	return nil
}

func connectAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("cluster id must be provided")
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
	ns, err := control.Node().List(ctx, cluster)
	if err != nil {
		return err
	}
	if len(ns) == 0 {
		return fmt.Errorf("No nodes found for %q", cluster)
	}

	err = nodes.WaitHealthy(ctx, ns)
	if err != nil {
		return err
	}

	return nodes.Connect(ctx, ns)
}
