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
	"github.com/Netflix/p2plab/experiments"
	"github.com/Netflix/p2plab/query"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli"
)

var experimentCommand = cli.Command{
	Name:    "experiment",
	Aliases: []string{"e"},
	Usage:   "Manage experiments.",
	Subcommands: []cli.Command{
		{
			Name:    "start",
			Aliases: []string{"s"},
			Usage:   "Starts an experiment",
			Action:  startExperimentAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "name",
					Usage: "Name of the experiment, by default takes the name of the experiment definition.",
				},
			},
		},
		{
			Name:    "inspect",
			Aliases: []string{"i"},
			Usage:   "Displays detailed information on a experiment.",
			Action:  inspectExperimentAction,
		},
		{
			Name:    "label",
			Aliases: []string{"l"},
			Usage:   "Add or remove labels from experiments.",
			Action:  labelExperimentsAction,
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name:  "add",
					Usage: "Adds a label.",
				},
				&cli.StringSliceFlag{
					Name:  "remove,rm",
					Usage: "Removes a label.",
				},
			},
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "List experiments.",
			Action:  listExperimentAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "query,q",
					Usage: "Runs a query to filter the listed experiments.",
				},
			},
		},
		{
			Name:    "remove",
			Aliases: []string{"rm"},
			Usage:   "Remove experiments.",
			Action:  removeExperimentsAction,
		},
	},
}

func startExperimentAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("experiment definition must be provided")
	}

	filename := c.Args().First()
	name := c.String("name")
	if name == "" {
		name = ExtractNameFromFilename(filename)
	}

	edef, err := experiments.Parse(filename)
	if err != nil {
		return err
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	experiment, err := control.Experiment().Start(CommandContext(c), name, edef)
	if err != nil {
		return err
	}

	log.Info().Msgf("Started experiment %q", experiment.Metadata().ID)
	return nil
}

func inspectExperimentAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("experiment id must be provided")
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	id := c.Args().First()
	experiment, err := control.Experiment().Get(CommandContext(c), id)
	if err != nil {
		return err
	}

	return CommandPrinter(c).Print(experiment.Metadata())
}

func labelExperimentsAction(c *cli.Context) error {
	var ids []string
	for i := 0; i < c.NArg(); i++ {
		ids = append(ids, c.Args().Get(i))
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	experiments, err := control.Experiment().Label(ctx, ids, c.StringSlice("add"), c.StringSlice("remove"))
	if err != nil {
		return err
	}

	l := make([]interface{}, len(experiments))
	for i, e := range experiments {
		l[i] = e.Metadata()
	}

	return CommandPrinter(c).Print(l)
}

func listExperimentAction(c *cli.Context) error {
	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	var opts []p2plab.ListOption
	if c.IsSet("query") {
		q, err := query.Parse(c.String("query"))
		if err != nil {
			return err
		}
		log.Debug().Msgf("Parsed query as %q", q)

		opts = append(opts, p2plab.WithQuery(q.String()))
	}

	experiments, err := control.Experiment().List(CommandContext(c), opts...)
	if err != nil {
		return err
	}

	l := make([]interface{}, len(experiments))
	for i, e := range experiments {
		l[i] = e.Metadata()
	}

	return CommandPrinter(c).Print(l)
}

func removeExperimentsAction(c *cli.Context) error {
	var ids []string
	for i := 0; i < c.NArg(); i++ {
		ids = append(ids, c.Args().Get(i))
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	err = control.Experiment().Remove(ctx, ids...)
	if err != nil {
		return err
	}

	return nil
}
