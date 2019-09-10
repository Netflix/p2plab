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
	"bytes"
	"context"
	"encoding/base64"
	"os"

	"github.com/Netflix/p2plab/labapp"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/cliutil"
	"github.com/Netflix/p2plab/pkg/traceutil"
	"github.com/Netflix/p2plab/version"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/urfave/cli"
)

func App(ctx context.Context) *cli.App {
	app := cli.NewApp()
	app.Name = "labapp"
	app.Version = version.Version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "root",
			Usage:  "path to state directory",
			Value:  "./tmp/labapp",
			EnvVar: "LABAPP_ROOT",
		},
		cli.StringFlag{
			Name:   "address,a",
			Usage:  "address for labapp's HTTP server",
			Value:  ":7003",
			EnvVar: "LABAPP_ADDRESS",
		},
		cli.StringFlag{
			Name:   "node-id",
			Usage:  "node id for the trace's service ID",
			EnvVar: "LABAPP_NODE_ID",
		},
		cli.StringFlag{
			Name:   "trace",
			Usage:  "base64 encoded opentracing trace injected as a binary carrier",
			EnvVar: "LABAPP_TRACE",
		},
		cli.IntFlag{
			Name:   "libp2p-port",
			Usage:  "port for libp2p",
			Value:  0,
			EnvVar: "LABAPP_LIBP2P_PORT",
		},
		cli.StringSliceFlag{
			Name:   "libp2p-transports",
			Usage:  "transports for libp2p [tcp, ws, quic]",
			EnvVar: "LABAPP_LIBP2P_TRANSPORTS",
		},
		cli.StringSliceFlag{
			Name:   "libp2p-muxers",
			Usage:  "muxers for libp2p [mplex, yamux]",
			EnvVar: "LABAPP_LIBP2P_MUXERS",
		},
		cli.StringSliceFlag{
			Name:   "libp2p-security-transports",
			Usage:  "security transports for libp2p [tls, secio, noise]",
			EnvVar: "LABAPP_LIBP2P_SECURITY_TRANSPORTS",
		},
		cli.StringFlag{
			Name:   "libp2p-routing",
			Usage:  "routing for libp2p [nil, kaddht]",
			EnvVar: "LABAPP_LIBP2P_ROUTING",
		},
		cli.StringFlag{
			Name:   "log-level,l",
			Usage:  "set the logging level [debug, info, warn, error, fatal, panic, none]",
			Value:  "debug",
			EnvVar: "LABAPP_LOG_LEVEL",
		},
	}
	app.Action = appAction

	// Setup context.
	cliutil.AttachAppContext(ctx, app)

	return app
}

func appAction(c *cli.Context) error {
	root := c.GlobalString("root")
	err := os.MkdirAll(root, 0711)
	if err != nil {
		return err
	}

	ctx := cliutil.CommandContext(c)
	ctx, tracer, closer := traceutil.New(ctx, "labapp", nil)
	defer closer.Close()

	if c.IsSet("trace") {
		trace, err := base64.StdEncoding.DecodeString(c.GlobalString("trace"))
		if err != nil {
			return err
		}

		spanContext, err := tracer.Extract(opentracing.Binary, bytes.NewReader(trace))
		if err != nil {
			return errors.Wrap(err, "failed to extract trace from --trace")
		}

		span := tracer.StartSpan(c.GlobalString("node-id"), opentracing.ChildOf(spanContext))
		defer span.Finish()
		ctx = opentracing.ContextWithSpan(ctx, span)
	}

	app, err := labapp.New(ctx, root, c.GlobalString("address"), c.GlobalInt("libp2p-port"), zerolog.Ctx(ctx), metadata.PeerDefinition{
		Transports:         c.GlobalStringSlice("libp2p-transports"),
		Muxers:             c.GlobalStringSlice("libp2p-muxers"),
		SecurityTransports: c.GlobalStringSlice("libp2p-security-transports"),
		Routing:            c.GlobalString("libp2p-routing"),
	})
	if err != nil {
		return err
	}
	defer app.Close()

	return app.Serve(ctx)
}
