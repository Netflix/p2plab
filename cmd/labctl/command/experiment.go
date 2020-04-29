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
	"fmt"
	"os"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/experiments"
	"github.com/Netflix/p2plab/pkg/cliutil"
	"github.com/Netflix/p2plab/printer"
	"github.com/Netflix/p2plab/query"
	"github.com/rs/zerolog"
	"github.com/urfave/cli"
)

var experimentCommand = cli.Command{
	Name:    "experiment",
	Aliases: []string{"e"},
	Usage:   "Manage experiments.",
	Subcommands: []cli.Command{
		{
			Name:      "create",
			Aliases:   []string{"c"},
			Usage:     "Creates an experiment from a definition file",
			ArgsUsage: "<filename>",
			Action:    createExperimentAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "output-format",
					Usage: "one of: json, csv",
					Value: "csv",
				},
				cli.StringFlag{
					Name:  "output-file",
					Usage: "file to write data to, only applicable to csv",
					Value: "reports.csv",
				},
				&cli.StringFlag{
					Name:  "name",
					Usage: "Name of the experiment, by default takes the name of the experiment definition.",
				},
				&cli.BoolFlag{
					Name:  "dry-run",
					Usage: "dry run the epxeriment creation, parsing the cue file and printing it to stdout",
				},
			},
		},
		{
			Name:      "inspect",
			Aliases:   []string{"i"},
			Usage:     "Displays detailed information on a experiment.",
			ArgsUsage: "<id>",
			Action:    inspectExperimentAction,
		},
		{
			Name:      "label",
			Aliases:   []string{"l"},
			Usage:     "Add or remove labels from experiments.",
			ArgsUsage: " ",
			Action:    labelExperimentsAction,
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
			Name:      "list",
			Aliases:   []string{"ls"},
			Usage:     "List experiments.",
			ArgsUsage: " ",
			Action:    listExperimentAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "query,q",
					Usage: "Runs a query to filter the listed experiments.",
				},
			},
		},
		{
			Name:      "remove",
			Aliases:   []string{"rm"},
			Usage:     "Remove experiments.",
			ArgsUsage: "[<id> ...]",
			Action:    removeExperimentsAction,
		},
	},
}

func createExperimentAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("experiment definition must be provided")
	}

	p, err := CommandPrinter(c, printer.OutputJSON)
	if err != nil {
		return err
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

	if c.Bool("dry-run") {
		dt, err := edef.ToJSON()
		if err != nil {
			return err
		}
		fmt.Println(string(dt))
		return nil
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := cliutil.CommandContext(c)
	id, err := control.Experiment().Create(ctx, name, edef)
	if err != nil {
		return err
	}

	experiment, err := control.Experiment().Get(ctx, id)
	if err != nil {
		return err
	}
	if c.String("output-format") == "csv" {
		fh, err := os.Create(c.String("output-file"))
		if err != nil {
			return err
		}
		defer fh.Close()
		return experiments.ReportToCSV(experiment.Metadata().Reports, fh)
	}
	zerolog.Ctx(ctx).Info().Msgf("Completed experiment %q", experiment.Metadata().ID)
	return p.Print(experiment.Metadata())
}

func inspectExperimentAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("experiment id must be provided")
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
	experiment, err := control.Experiment().Get(ctx, id)
	if err != nil {
		return err
	}

	return p.Print(experiment.Metadata())
}

func labelExperimentsAction(c *cli.Context) error {
	p, err := CommandPrinter(c, printer.OutputTable)
	if err != nil {
		return err
	}

	var ids []string
	for i := 0; i < c.NArg(); i++ {
		ids = append(ids, c.Args().Get(i))
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := cliutil.CommandContext(c)
	experiments, err := control.Experiment().Label(ctx, ids, c.StringSlice("add"), c.StringSlice("remove"))
	if err != nil {
		return err
	}

	l := make([]interface{}, len(experiments))
	for i, e := range experiments {
		l[i] = e.Metadata()
	}

	return p.Print(l)
}

func listExperimentAction(c *cli.Context) error {
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

	experiments, err := control.Experiment().List(ctx, opts...)
	if err != nil {
		return err
	}

	l := make([]interface{}, len(experiments))
	for i, e := range experiments {
		l[i] = e.Metadata()
	}

	return p.Print(l)
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

	ctx := cliutil.CommandContext(c)
	err = control.Experiment().Remove(ctx, ids...)
	if err != nil {
		return err
	}

	return nil
}
