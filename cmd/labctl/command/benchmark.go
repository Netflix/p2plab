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
			Usage:   "Start benchmark for a scenario.",
			Action:  startBenchmarkAction,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "no-reset",
					Usage: "Skips resetting the cluster to maintain a stale state",
				},
			},
		},
		{
			Name:    "cancel",
			Aliases: []string{"c"},
			Usage:   "Cancel a running benchmark.",
			Action:  cancelBenchmarkAction,
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "List benchmarks",
			Action:  listBenchmarkAction,
		},
		{
			Name:    "report",
			Aliases: []string{"r"},
			Usage:   "Shows a benchmark report.",
			Action:  reportBenchmarkAction,
		},
		{
			Name:   "logs",
			Usage:  "Shows logs of a benchmark.",
			Action: logBenchmarkAction,
		},
	},
}

func startBenchmarkAction(c *cli.Context) error {
	if c.NArg() != 2 {
		return errors.New("cluster id and scenario name must be provided")
	}

	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	cluster, scenario := c.Args().Get(0), c.Args().Get(1)

	var opts []p2plab.StartBenchmarkOption
	if c.Bool("no-reset") {
		opts = append(opts, p2plab.WithBenchmarkNoReset())
	}

	_, err = cln.Benchmark().Start(ctx, cluster, scenario, opts...)
	if err != nil {
		return err
	}

	return nil
}

func cancelBenchmarkAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("benchmark id must be provided")
	}

	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	benchmark, err := cln.Benchmark().Get(ctx, c.Args().First())
	if err != nil {
		return err
	}

	err = benchmark.Cancel(ctx)
	if err != nil {
		return err
	}

	log.Info().Msgf("Cancelled benchmark %q", benchmark.Metadata().ID)
	return nil
}

func listBenchmarkAction(c *cli.Context) error {
	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	benchmarks, err := cln.Benchmark().List(ctx)
	if err != nil {
		return err
	}

	l := make([]interface{}, len(benchmarks))
	for i, b := range benchmarks {
		l[i] = b.Metadata()
	}

	return CommandPrinter(c).Print(l)
}

func reportBenchmarkAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("benchmark id must be provided")
	}

	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	benchmark, err := cln.Benchmark().Get(ctx, c.Args().First())
	if err != nil {
		return err
	}

	report, err := benchmark.Report(ctx)
	if err != nil {
		return err
	}

	return CommandPrinter(c).Print(report)
}

func logBenchmarkAction(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("benchmark id must be provided")
	}

	cln, err := ResolveClient(c)
	if err != nil {
		return err
	}

	ctx := CommandContext(c)
	benchmark, err := cln.Benchmark().Get(ctx, c.Args().First())
	if err != nil {
		return err
	}

	logs, err := benchmark.Logs(ctx)
	if err != nil {
		return err
	}
	defer logs.Close()

	// TODO: Stream logs
	return CommandPrinter(c).Print(logs)
}
