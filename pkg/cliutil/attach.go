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

package cliutil

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/urfave/cli"
)

func AttachAppContext(ctx context.Context, app *cli.App) {
	before := app.Before
	app.Before = func(c *cli.Context) error {
		if before != nil {
			if err := before(c); err != nil {
				return err
			}
		}

		level, err := zerolog.ParseLevel(c.GlobalString("log-level"))
		if err != nil {
			return err
		}

		rootLogger := zerolog.New(os.Stderr).Level(level).With().Timestamp().Logger()
		logger := &rootLogger
		ctx = logger.WithContext(ctx)
		c.App.Metadata["context"] = ctx
		return nil
	}
}

func CommandContext(c *cli.Context) context.Context {
	return c.App.Metadata["context"].(context.Context)
}

func JoinBefore(fns ...cli.BeforeFunc) cli.BeforeFunc {
	return func(c *cli.Context) error {
		for _, fn := range fns {
			if fn == nil {
				continue
			}

			err := fn(c)
			if err != nil {
				return err
			}
		}
		return nil
	}
}
