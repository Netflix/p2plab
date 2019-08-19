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
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli"
)

var benchmarkCommand = cli.Command{
	Name:    "benchmark",
	Aliases: []string{"b"},
	Usage:   "Manage benchmarks.",
	Subcommands: []cli.Command{
		{
			Name:    "start",
			Aliases: []string{"s"},
			Usage:   "Start benchmark for a benchmark.",
			Action:  startBenchmarkAction,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "no-reset",
					Usage: "Skips resetting the cluster to maintain a stale state",
				},
			},
		},
		{
			Name:    "inspect",
			Aliases: []string{"i"},
			Usage:   "Displays detailed information on a benchmark.",
			Action:  inspectBenchmarkAction,
		},
		{
			Name:    "label",
			Aliases: []string{"l"},
			Usage:   "Add or remove labels from benchmarks.",
			Action:  labelBenchmarksAction,
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
			Usage:   "List benchmarks",
			Action:  listBenchmarkAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "query,q",
					Usage: "Runs a query to filter the listed benchmarks.",
				},
			},
		},
		{
			Name:    "remove",
			Aliases: []string{"rm"},
			Usage:   "Remove benchmarks.",
			Action:  removeBenchmarksAction,
		},
	},
}

func startBenchmarkAction(c *cli.Context) error {
	if c.NArg() != 2 {
		return errors.New("cluster id and benchmark name must be provided")
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	cluster, benchmark := c.Args().Get(0), c.Args().Get(1)

	var opts []p2plab.StartBenchmarkOption
	if c.Bool("no-reset") {
		opts = append(opts, p2plab.WithBenchmarkNoReset())
	}

	_, err = control.Benchmark().Start(ctx, cluster, benchmark, opts...)
	if err != nil {
		return err
	}

	return nil
}

func inspectBenchmarkAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("benchmark id must be provided")
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	id := c.Args().First()
	benchmark, err := control.Benchmark().Get(CommandContext(c), id)
	if err != nil {
		return err
	}

	return CommandPrinter(c).Print(benchmark.Metadata())
}

func labelBenchmarksAction(c *cli.Context) error {
	var ids []string
	for i := 0; i < c.NArg(); i++ {
		ids = append(ids, c.Args().Get(i))
	}

	control, err := ResolveControl(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	benchmarks, err := control.Benchmark().Label(ctx, ids, c.StringSlice("add"), c.StringSlice("remove"))
	if err != nil {
		return err
	}

	l := make([]interface{}, len(benchmarks))
	for i, b := range benchmarks {
		l[i] = b.Metadata()
	}

	return CommandPrinter(c).Print(l)
}

func listBenchmarkAction(c *cli.Context) error {
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

	ctx := CommandContext(c)
	benchmarks, err := control.Benchmark().List(ctx, opts...)
	if err != nil {
		return err
	}

	l := make([]interface{}, len(benchmarks))
	for i, b := range benchmarks {
		l[i] = b.Metadata()
	}

	return CommandPrinter(c).Print(l)
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

	ctx := CommandContext(c)
	err = control.Benchmark().Remove(ctx, ids...)
	if err != nil {
		return err
	}

	return nil
}
