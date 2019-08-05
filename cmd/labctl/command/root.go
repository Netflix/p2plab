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

	"github.com/Netflix/p2plab/version"
	"github.com/urfave/cli"
)

func App(ctx context.Context) *cli.App {
	app := cli.NewApp()
	app.Name = "labctl"
	app.Version = version.Version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "address,a",
			Usage: "address for labd",
			Value: "http://127.0.0.1:7001",
		},
		cli.StringFlag{
			Name:  "log-level,l",
			Usage: "set the logging level [debug, info, warn, error, fatal, panic]",
		},
		cli.StringFlag{
			Name:  "output,o",
			Usage: "set the output printer [unix, json]",
			Value: "unix",
		},
	}
	app.Commands = []cli.Command{
		clusterCommand,
		nodeCommand,
		scenarioCommand,
		benchmarkCommand,
	}

	// Setup tracers and context.
	AttachAppContext(ctx, app)

	// Setup output printer.
	AttachAppPrinter(app)

	return app
}
