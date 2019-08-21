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
	"os"

	"github.com/Netflix/p2plab/labd"
	"github.com/Netflix/p2plab/pkg/cliutil"
	"github.com/Netflix/p2plab/version"
	"github.com/rs/zerolog"
	"github.com/urfave/cli"
)

func App(ctx context.Context) *cli.App {
	app := cli.NewApp()
	app.Name = "labd"
	app.Version = version.Version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "root",
			Usage: "path to state directory",
			Value: "./tmp/labd",
			EnvVar: "LABD_ROOT",
		},
		cli.StringFlag{
			Name:  "address,a",
			Usage: "address for labd's HTTP server",
			Value: ":7001",
			EnvVar: "LABD_ADDRESS",
		},
		cli.StringFlag{
			Name:  "log-level,l",
			Usage: "set the logging level [debug, info, warn, error, fatal, panic]",
			Value: "debug",
			EnvVar: "LABD_LOG_LEVEL",
		},
	}
	app.Action = daemonAction

	// Setup context.
	cliutil.AttachAppContext(ctx, app)

	return app
}

func daemonAction(c *cli.Context) error {
	root := c.GlobalString("root")
	err := os.MkdirAll(root, 0711)
	if err != nil {
		return err
	}

	ctx := cliutil.CommandContext(c)
	daemon, err := labd.New(root, c.String("address"), zerolog.Ctx(ctx))
	if err != nil {
		return err
	}
	defer daemon.Close()

	return daemon.Serve(ctx)
}
