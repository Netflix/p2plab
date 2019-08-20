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
	"github.com/Netflix/p2plab/scenarios"
	"github.com/rs/zerolog"
	"github.com/urfave/cli"
)

var scenarioCommand = cli.Command{
	Name:    "scenario",
	Aliases: []string{"s"},
	Usage:   "Manage scenarios.",
	Subcommands: []cli.Command{
		{
			Name:    "create",
			Aliases: []string{"c"},
			Usage:   "Creates a new scenario.",
			Action:  createScenarioAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "name",
					Usage: "Name of the scenario, by default takes the name of the scenario definition.",
				},
			},
		},
		{
			Name:    "inspect",
			Aliases: []string{"i"},
			Usage:   "Displays detailed information on a scenario.",
			Action:  inspectScenarioAction,
		},
		{
			Name:    "label",
			Aliases: []string{"l"},
			Usage:   "Add or remove labels from scenarios.",
			Action:  labelScenariosAction,
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
			Usage:   "List scenarios.",
			Action:  listScenarioAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "query,q",
					Usage: "Runs a query to filter the listed scenarios.",
				},
			},
		},
		{
			Name:    "remove",
			Aliases: []string{"rm"},
			Usage:   "Remove scenarios.",
			Action:  removeScenariosAction,
		},
	},
}

func createScenarioAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("scenario definition must be provided")
	}

	filename := c.Args().First()
	name := c.String("name")
	if name == "" {
		name = ExtractNameFromFilename(filename)
	}

	sdef, err := scenarios.Parse(filename)
	if err != nil {
		return err
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	scenario, err := control.Scenario().Create(ctx, name, sdef)
	if err != nil {
		return err
	}

	zerolog.Ctx(ctx).Info().Msgf("Created scenario %q", scenario.Metadata().ID)
	return nil
}

func inspectScenarioAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("scenario id must be provided")
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	name := c.Args().Get(0)
	scenario, err := control.Scenario().Get(CommandContext(c), name)
	if err != nil {
		return err
	}

	return CommandPrinter(c).Print(scenario.Metadata())
}

func labelScenariosAction(c *cli.Context) error {
	var names []string
	for i := 0; i < c.NArg(); i++ {
		names = append(names, c.Args().Get(i))
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	scenarios, err := control.Scenario().Label(ctx, names, c.StringSlice("add"), c.StringSlice("remove"))
	if err != nil {
		return err
	}

	l := make([]interface{}, len(scenarios))
	for i, s := range scenarios {
		l[i] = s.Metadata()
	}

	return CommandPrinter(c).Print(l)
}

func listScenarioAction(c *cli.Context) error {
	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	var opts []p2plab.ListOption
	ctx := CommandContext(c)
	if c.IsSet("query") {
		q, err := query.Parse(ctx, c.String("query"))
		if err != nil {
			return err
		}

		opts = append(opts, p2plab.WithQuery(q.String()))
	}

	scenarios, err := control.Scenario().List(ctx, opts...)
	if err != nil {
		return err
	}

	l := make([]interface{}, len(scenarios))
	for i, s := range scenarios {
		l[i] = s.Metadata()
	}

	return CommandPrinter(c).Print(l)
}

func removeScenariosAction(c *cli.Context) error {
	var names []string
	for i := 0; i < c.NArg(); i++ {
		names = append(names, c.Args().Get(i))
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	err = control.Scenario().Remove(ctx, names...)
	if err != nil {
		return err
	}

	zerolog.Ctx(ctx).Info().Strs("names", names).Msg("Removed scenarios")
	return nil
}
