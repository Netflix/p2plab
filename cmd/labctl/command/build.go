// Copyright 2019 Netflix, Inc.
//
// Licebuildsed under the Apache Licebuildse, Version 2.0 (the "Licebuildse");
// you may not use this file except in compliance with the Licebuildse.
// You may obtain a copy of the Licebuildse at
//
//     http://www.apache.org/licebuildses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the Licebuildse is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the Licebuildse for the specific language governing permissiobuilds and
// limitatiobuilds under the Licebuildse.

package command

import (
	"io"
	"os"

	"github.com/Netflix/p2plab/pkg/cliutil"
	"github.com/Netflix/p2plab/printer"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var buildCommand = cli.Command{
	Name:    "build",
	Aliases: []string{"b"},
	Usage:   "Manage builds.",
	Subcommands: []cli.Command{
		{
			Name:      "inspect",
			Aliases:   []string{"i"},
			Usage:     "Displays detailed information on a build.",
			ArgsUsage: "<id>",
			Action:    inspectBuildAction,
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "List builds.",
			Action:  listBuildAction,
		},
		{
			Name:      "upload",
			Aliases:   []string{"u"},
			ArgsUsage: "<file>",
			Usage:     "Uploads a binary to use as a build.",
			Action:    uploadBuildAction,
		},
		{
			Name:      "download",
			Aliases:   []string{"d"},
			ArgsUsage: "<id> <file>",
			Usage:     "Downloads a build's binary to a file.",
			Action:    downloadBuildAction,
		},
	},
}

func inspectBuildAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("build id must be provided")
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
	id := c.Args().First()
	build, err := control.Build().Get(ctx, id)
	if err != nil {
		return err
	}

	return p.Print(build.Metadata())
}

func listBuildAction(c *cli.Context) error {
	p, err := CommandPrinter(c, printer.OutputTable)
	if err != nil {
		return err
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := cliutil.CommandContext(c)
	builds, err := control.Build().List(ctx)
	if err != nil {
		return err
	}

	l := make([]interface{}, len(builds))
	for i, n := range builds {
		l[i] = n.Metadata()
	}

	return p.Print(l)
}

func uploadBuildAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("path to binary must be provided")
	}

	p, err := CommandPrinter(c, printer.OutputJSON)
	if err != nil {
		return err
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	f, err := os.Open(c.Args().First())
	if err != nil {
		return err
	}
	defer f.Close()

	ctx := cliutil.CommandContext(c)
	build, err := control.Build().Upload(ctx, f)
	if err != nil {
		return err
	}

	return p.Print(build.Metadata())
}

func downloadBuildAction(c *cli.Context) error {
	if c.NArg() != 2 {
		return errors.New("id and destination path must be provided")
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := cliutil.CommandContext(c)
	id := c.Args().First()
	build, err := control.Build().Get(ctx, id)
	if err != nil {
		return err
	}

	rc, err := build.Open(ctx)
	if err != nil {
		return err
	}
	defer rc.Close()

	dest := c.Args().Get(1)
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, rc)
	if err != nil {
		return err
	}

	return os.Chmod(dest, 0775)
}
