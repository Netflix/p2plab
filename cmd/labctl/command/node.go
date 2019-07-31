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
	},
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
