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
	"os"

	"github.com/Netflix/p2plab/downloaders"
	"github.com/Netflix/p2plab/downloaders/s3downloader"
	"github.com/Netflix/p2plab/labagent"
	"github.com/Netflix/p2plab/pkg/cliutil"
	"github.com/Netflix/p2plab/providers/terraform"
	"github.com/Netflix/p2plab/version"
	"github.com/rs/zerolog"
	"github.com/urfave/cli"
)

func App(ctx context.Context) *cli.App {
	app := cli.NewApp()
	app.Name = "labagent"
	app.Version = version.Version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "root",
			Usage:  "path to state directory",
			Value:  "./tmp/labagent",
			EnvVar: "LABAGENT_ROOT",
		},
		cli.StringFlag{
			Name:   "address,a",
			Usage:  "address for labd's HTTP server",
			Value:  fmt.Sprintf(":%d", terraform.DefaultAgentPort),
			EnvVar: "LABAGENT_ADDRESS",
		},
		cli.StringFlag{
			Name:   "app-root",
			Usage:  "path to labapp's state directory",
			Value:  "./tmp/labapp",
			EnvVar: "LABAGENT_APP_ROOT",
		},
		cli.StringFlag{
			Name:   "app-address",
			Usage:  "address for labapp's HTTP server",
			Value:  fmt.Sprintf("http://localhost:%d", terraform.DefaultAppPort),
			EnvVar: "LABAGENT_APP_ADDRESS",
		},
		cli.StringFlag{
			Name:   "log-level,l",
			Usage:  "set the logging level [debug, info, warn, error, fatal, panic]",
			Value:  "debug",
			EnvVar: "LABAGENT_LOG_LEVEL",
		},
		cli.StringFlag{
			Name:   "downloader.s3.region",
			Usage:  "region for s3 downloader",
			EnvVar: "LABAGENT_DOWNLOADER_S3_REGION",
		},
	}
	app.Action = agentAction

	// Setup context.
	cliutil.AttachAppContext(ctx, app)

	return app
}

func agentAction(c *cli.Context) error {
	root := c.GlobalString("root")
	err := os.MkdirAll(root, 0711)
	if err != nil {
		return err
	}

	ctx := cliutil.CommandContext(c)
	agent, err := labagent.New(root, c.String("address"), c.String("app-root"), c.String("app-address"), zerolog.Ctx(ctx),
		labagent.WithDownloaderSettings(downloaders.DownloaderSettings{
			S3: s3downloader.S3DownloaderSettings{
				Region: c.String("downloader.s3.region"),
			},
		}),
	)
	if err != nil {
		return err
	}
	defer agent.Close()

	return agent.Serve(ctx)
}
