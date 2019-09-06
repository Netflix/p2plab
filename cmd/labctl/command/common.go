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
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Netflix/p2plab/errdefs"
	"github.com/Netflix/p2plab/pkg/cliutil"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/Netflix/p2plab/pkg/logutil"
	"github.com/Netflix/p2plab/pkg/traceutil"
	"github.com/Netflix/p2plab/printer"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/urfave/cli"
)

func AttachAppContext(ctx context.Context, app *cli.App) {
	var (
		logger *zerolog.Logger
		writer io.Writer
		tracer opentracing.Tracer
		closer io.Closer
		span   opentracing.Span
	)

	before := app.Before
	app.Before = func(c *cli.Context) error {
		if before != nil {
			if err := before(c); err != nil {
				return err
			}
		}

		var err error
		logger, writer, err = newLogger(c)
		if err != nil {
			return err
		}

		ctx, tracer, closer = traceutil.New(ctx, "labctl", nil)
		return nil
	}

	for i, cmd := range app.Commands {
		for j, subcmd := range cmd.Subcommands {
			func(before cli.BeforeFunc) {
				name := strings.Join([]string{cmd.Name, subcmd.Name}, " ")
				app.Commands[i].Subcommands[j].Before = func(c *cli.Context) error {
					if before != nil {
						if err := before(c); err != nil {
							return err
						}
					}

					span = tracer.StartSpan(name)
					span.SetTag("command", strings.Join(os.Args, " "))

					ctx = logger.WithContext(ctx)
					ctx = logutil.WithLogWriter(ctx, writer)
					ctx = opentracing.ContextWithSpan(ctx, span)

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

		if span != nil {
			span.Finish()
		}
		return closer.Close()
	}
}

func AttachAppClient(app *cli.App) {
	app.Before = cliutil.JoinBefore(app.Before, func(c *cli.Context) error {
		var opts []httputil.ClientOption
		if c.GlobalString("log-level") == "debug" {
			logger, _, err := newLogger(c)
			if err != nil {
				return err
			}

			opts = append(opts, httputil.WithLogger(logger))
		}

		client, err := httputil.NewClient(httputil.NewHTTPClient(), opts...)
		if err != nil {
			return err
		}

		app.Metadata["client"] = client
		return nil
	})
}

func CommandPrinter(c *cli.Context, auto printer.OutputType) (printer.Printer, error) {
	return printer.GetPrinter(printer.OutputType(c.GlobalString("output")), auto)
}

func CommandClient(c *cli.Context) *httputil.Client {
	return c.App.Metadata["client"].(*httputil.Client)
}

func newLogger(c *cli.Context) (*zerolog.Logger, io.Writer, error) {
	var out io.Writer
	switch c.GlobalString("log-writer") {
	case "console":
		out = zerolog.ConsoleWriter{Out: os.Stderr}
	case "json":
		out = os.Stderr
	default:
		return nil, nil, errors.Wrapf(errdefs.ErrInvalidArgument, "unknown log writer %q", c.GlobalString("log-writer"))
	}

	level, err := zerolog.ParseLevel(c.GlobalString("log-level"))
	if err != nil {
		return nil, nil, err
	}

	logger := zerolog.New(out).
		Level(level).
		With().Timestamp().Logger()

	return &logger, out, nil
}

func ExtractNameFromFilename(filename string) string {
	return strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
}
