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
