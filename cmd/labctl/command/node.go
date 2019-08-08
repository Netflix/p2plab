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
	"github.com/Netflix/p2plab/metadata"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var nodeCommand = cli.Command{
	Name:    "node",
	Aliases: []string{"n"},
	Usage:   "Manage nodes.",
	Subcommands: []cli.Command{
		{
			Name:   "ssh",
			Usage:  "SSH into a node.",
			Action: sshNodeAction,
		},
		{
			Name:   "run",
			Usage:  "Runs a task on a node.",
			Action: runNodeAction,
		},
	},
}

func sshNodeAction(c *cli.Context) error {
	if c.NArg() != 2 {
		return errors.New("cluster id and node id must be provided")
	}

	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	node, err := cln.Node().Get(ctx, c.Args().Get(0), c.Args().Get(1))
	if err != nil {
		return err
	}

	err = node.SSH(ctx)
	if err != nil {
		return err
	}

	return nil
}

func runNodeAction(c *cli.Context) error {
	if c.NArg() < 3 {
		return errors.New("cluster id, node id, task type must be provided")
	}

	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	node, err := cln.Node().Get(ctx, c.Args().Get(0), c.Args().Get(1))
	if err != nil {
		return err
	}

	var task metadata.Task

	taskType := metadata.TaskType(c.Args().Get(2))
	switch taskType {
	case metadata.TaskGet, metadata.TaskUpdate:
		task.Type = taskType
	default:
		return errors.Errorf("unrecognized task type: %q", taskType)
	}

	if c.NArg() == 4 {
		task.Target = c.Args().Get(3)
	}

	err = node.Run(ctx, task)
	if err != nil {
		return err
	}

	return nil
}
