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

	"github.com/Netflix/p2plab/metadata"
	"github.com/urfave/cli"
)

var debugCommand = cli.Command{
	Name:    "debug",
	Aliases: []string{"d"},
	Usage:   "Debugging tools.",
	Hidden:  true,
	Subcommands: []cli.Command{
		{
			Name:    "peer",
			Aliases: []string{"s"},
			Usage:   "Retrieves the peer info from a labapp",
			Action:  peerInfoAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "address",
					Usage: "address for labapp's HTTP server",
					Value: "http://localhost:7002",
				},
			},
		},
		{
			Name:    "run",
			Aliases: []string{"s"},
			Usage:   "Runs a task on a labapp.",
			Action:  runTaskAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "address",
					Usage: "address for labapp's HTTP server",
					Value: "http://localhost:7002",
				},
			},
		},
	},
}

func peerInfoAction(c *cli.Context) error {
	app, err := ResolveApplication(c, c.String("address"))
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
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

	app, err := ResolveApplication(c, c.String("address"))
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	err = app.Run(ctx, metadata.Task{
		Type:    metadata.TaskType(c.Args().Get(0)),
		Subject: c.Args().Get(1),
	})
	if err != nil {
		return err
	}

	return nil
}
