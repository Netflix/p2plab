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

	"github.com/Netflix/p2plab/scenarios"
	"github.com/rs/zerolog/log"
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
			Name:    "remove",
			Aliases: []string{"rm"},
			Usage:   "Remove scenarios.",
			Action:  removeScenarioAction,
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "List scenarios.",
			Action:  listScenarioAction,
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

	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	scenario, err := cln.Scenario().Create(ctx, name, sdef)
	if err != nil {
		return err
	}

	log.Info().Msgf("Created scenario %q", scenario.Metadata().ID)
	return nil
}

func removeScenarioAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("scenario id must be provided")
	}

	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	scenario, err := cln.Scenario().Get(ctx, c.Args().First())
	if err != nil {
		return err
	}

	err = scenario.Remove(ctx)
	if err != nil {
		return err
	}

	log.Info().Msgf("Removed scenario %q", scenario.Metadata().ID)
	return nil
}

func listScenarioAction(c *cli.Context) error {
	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	scenarios, err := cln.Scenario().List(ctx)
	if err != nil {
		return err
	}

	l := make([]interface{}, len(scenarios))
	for i, s := range scenarios {
		l[i] = s.Metadata()
	}

	return CommandPrinter(c).Print(l)
}
