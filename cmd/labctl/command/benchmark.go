package command

import "github.com/urfave/cli"

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
		},
		{
			Name:    "cancel",
			Aliases: []string{"c"},
			Usage:   "Cancel a benchmark.",
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
			Name:   "log",
			Usage:  "Shows logs of a benchmark.",
			Action: logBenchmarkAction,
		},
	},
}

func startBenchmarkAction(c *cli.Context) error {
	return nil
}

func cancelBenchmarkAction(c *cli.Context) error {
	return nil
}

func listBenchmarkAction(c *cli.Context) error {
	return nil
}

func reportBenchmarkAction(c *cli.Context) error {
	return nil
}

func logBenchmarkAction(c *cli.Context) error {
	return nil
}
