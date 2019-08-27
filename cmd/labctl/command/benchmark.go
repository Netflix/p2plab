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
	"github.com/Netflix/p2plab/pkg/cliutil"
	"github.com/Netflix/p2plab/printer"
	"github.com/Netflix/p2plab/query"
	"github.com/rs/zerolog"
	"github.com/urfave/cli"
)

var benchmarkCommand = cli.Command{
	Name:    "benchmark",
	Aliases: []string{"b"},
	Usage:   "Manage benchmarks.",
	Subcommands: []cli.Command{
		{
			Name:      "create",
			Aliases:   []string{"s"},
			Usage:     "Benchmarks a scenario on a cluster.",
			ArgsUsage: "<cluster> <scenario>",
			Action:    createBenchmarkAction,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "no-reset",
					Usage: "Skips resetting the cluster to maintain a stale state",
				},
			},
		},
		{
			Name:      "inspect",
			Aliases:   []string{"i"},
			Usage:     "Displays detailed information on a benchmark.",
			ArgsUsage: "<id>",
			Action:    inspectBenchmarkAction,
		},
		{
			Name:      "label",
			Aliases:   []string{"l"},
			Usage:     "Add or remove labels from benchmarks.",
			ArgsUsage: " ",
			Action:    labelBenchmarksAction,
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
			Usage:     "List benchmarks",
			ArgsUsage: " ",
			Action:    listBenchmarkAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "query,q",
					Usage: "Runs a query to filter the listed benchmarks.",
				},
			},
		},
		{
			Name:      "report",
			Aliases:   []string{"r"},
			Usage:     "Display a benchmark's report.",
			ArgsUsage: "<id>",
			Action:    benchmarkReportAction,
		},
		{
			Name:      "remove",
			Aliases:   []string{"rm"},
			Usage:     "Remove benchmarks.",
			ArgsUsage: "[<id> ...]",
			Action:    removeBenchmarksAction,
		},
	},
}

func createBenchmarkAction(c *cli.Context) error {
	if c.NArg() != 2 {
		return errors.New("cluster and scenario name must be provided")
	}

	p, err := CommandPrinter(c, printer.OutputTable)
	if err != nil {
		return err
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := cliutil.CommandContext(c)
	cluster, scenario := c.Args().Get(0), c.Args().Get(1)

	var opts []p2plab.StartBenchmarkOption
	if c.Bool("no-reset") {
		opts = append(opts, p2plab.WithBenchmarkNoReset())
	}

	id, err := control.Benchmark().Create(ctx, cluster, scenario, opts...)
	if err != nil {
		return err
	}

	benchmark, err := control.Benchmark().Get(ctx, id)
	if err != nil {
		return err
	}
	zerolog.Ctx(ctx).Info().Msgf("Completed benchmark %q", benchmark.Metadata().ID)

	report, err := benchmark.Report(ctx)
	if err != nil {
		return err
	}

	return p.Print(report)
}

func inspectBenchmarkAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("benchmark id must be provided")
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
	benchmark, err := control.Benchmark().Get(ctx, id)
	if err != nil {
		return err
	}

	return p.Print(benchmark.Metadata())
}

func labelBenchmarksAction(c *cli.Context) error {
	var ids []string
	for i := 0; i < c.NArg(); i++ {
		ids = append(ids, c.Args().Get(i))
	}

	p, err := CommandPrinter(c, printer.OutputTable)
	if err != nil {
		return err
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := cliutil.CommandContext(c)
	benchmarks, err := control.Benchmark().Label(ctx, ids, c.StringSlice("add"), c.StringSlice("remove"))
	if err != nil {
		return err
	}

	l := make([]interface{}, len(benchmarks))
	for i, b := range benchmarks {
		l[i] = b.Metadata()
	}

	return p.Print(l)
}

func listBenchmarkAction(c *cli.Context) error {
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

	benchmarks, err := control.Benchmark().List(ctx, opts...)
	if err != nil {
		return err
	}

	l := make([]interface{}, len(benchmarks))
	for i, b := range benchmarks {
		l[i] = b.Metadata()
	}

	return p.Print(l)
}

func benchmarkReportAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("benchmark id must be provided")
	}

	p, err := CommandPrinter(c, printer.OutputTable)
	if err != nil {
		return err
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := cliutil.CommandContext(c)
	id := c.Args().First()
	benchmark, err := control.Benchmark().Get(ctx, id)
	if err != nil {
		return err
	}

	report, err := benchmark.Report(ctx)
	if err != nil {
		return err
	}

	return p.Print(report)
}

func removeBenchmarksAction(c *cli.Context) error {
	var ids []string
	for i := 0; i < c.NArg(); i++ {
		ids = append(ids, c.Args().Get(i))
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := cliutil.CommandContext(c)
	err = control.Benchmark().Remove(ctx, ids...)
	if err != nil {
		return err
	}

	return nil
}
