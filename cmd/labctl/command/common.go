package command

import (
	"context"
	"fmt"

	"github.com/Netflix/p2plab/printer"
	"github.com/urfave/cli"
)

type OutputType string

var (
	OutputTable OutputType = "table"
	OutputJSON  OutputType = "json"
)

func AttachAppContext(app *cli.App) {
	ctx := context.Background()

	for i, cmd := range app.Commands {
		for j, subcmd := range cmd.Subcommands {
			func(before cli.BeforeFunc) {
				// name := subcmd.Name
				app.Commands[i].Subcommands[j].Before = func(c *cli.Context) error {
					if before != nil {
						if err := before(c); err != nil {
							return err
						}
					}

					// Start span for context.
					c.App.Metadata["context"] = ctx
					return nil
				}
			}(subcmd.Before)
		}
	}

	after := app.After
	app.After = func(c *cli.Context) error {
		if after != nil {
			if err := after(c); err != nil {
				return err
			}
		}
		// Finish spans.
		return nil
	}

}

func AttachAppPrinter(app *cli.App) {
	app.Before = func(c *cli.Context) error {
		output := OutputType(c.String("output"))

		var p printer.Printer
		switch output {
		case OutputTable:
			p = printer.NewTablePrinter()
		case OutputJSON:
			p = printer.NewJSONPrinter()
		default:
			return fmt.Errorf("output %q is not valid", output)
		}

		c.App.Metadata["printer"] = p
		return nil
	}
}

func CommandContext(c *cli.Context) context.Context {
	return c.App.Metadata["context"].(context.Context)
}

func CommandPrinter(c *cli.Context) printer.Printer {
	return c.App.Metadata["printer"].(printer.Printer)
}
