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

	"github.com/Netflix/p2plab/experiments"
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
			Name:    "cancel",
			Aliases: []string{"c"},
			Usage:   "Cancels a running experiments.",
			Action:  cancelExperimentAction,
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "List experiments.",
			Action:  listExperimentAction,
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

func cancelExperimentAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("experiment id must be provided")
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	experiment, err := control.Experiment().Get(ctx, c.Args().First())
	if err != nil {
		return err
	}

	err = experiment.Cancel(ctx)
	if err != nil {
		return err
	}

	log.Info().Msgf("Cancelled experiment %q", experiment.Metadata().ID)
	return nil
}

func listExperimentAction(c *cli.Context) error {
	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	experiments, err := control.Experiment().List(CommandContext(c))
	if err != nil {
		return err
	}

	l := make([]interface{}, len(experiments))
	for i, e := range experiments {
		l[i] = e.Metadata()
	}

	return CommandPrinter(c).Print(l)
}
