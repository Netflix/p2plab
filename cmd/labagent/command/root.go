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

	"github.com/Netflix/p2plab/labagent"
	"github.com/Netflix/p2plab/version"
	"github.com/urfave/cli"
)

func App() *cli.App {
	app := cli.NewApp()
	app.Name = "labagent"
	app.Version = version.Version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "root",
			Usage: "path to state directory",
			Value: "./tmp/labagent",
		},
		cli.StringFlag{
			Name:  "address,a",
			Usage: "address for labd's HTTP server",
			Value: ":7002",
		},
		cli.StringFlag{
			Name:  "app-root",
			Usage: "path to labapp's state directory",
			Value: "./tmp/labapp",
		},
		cli.StringFlag{
			Name:  "app-address,a",
			Usage: "address for labapp's HTTP server",
			Value: ":7002",
		},
		cli.StringFlag{
			Name:  "log-level,l",
			Usage: "set the logging level [debug, info, warn, error, fatal, panic]",
		},
	}
	app.Action = agentAction
	return app
}

func agentAction(c *cli.Context) error {
	ctx := context.Background()

	root := c.GlobalString("root")
	err := os.MkdirAll(root, 0711)
	if err != nil {
		return err
	}

	agent, err := labagent.New(root, c.String("address"), c.String("app-root"), c.String("app-address"))
	if err != nil {
		return err
	}

	return agent.Serve(ctx)
}
