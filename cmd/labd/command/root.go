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
	"github.com/Netflix/p2plab/uploaders"
	"github.com/Netflix/p2plab/uploaders/fileuploader"
	"github.com/Netflix/p2plab/uploaders/s3uploader"
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
			Name:   "root",
			Usage:  "path to state directory",
			Value:  "./tmp/labd",
			EnvVar: "LABD_ROOT",
		},
		cli.StringFlag{
			Name:   "address,a",
			Usage:  "address for labd's HTTP server",
			Value:  ":7001",
			EnvVar: "LABD_ADDRESS",
		},
		cli.IntFlag{
			Name:   "libp2p-port",
			Usage:  "port for libp2p",
			Value:  0,
			EnvVar: "LABD_LIBP2P_PORT",
		},
		cli.StringFlag{
			Name:   "log-level,l",
			Usage:  "set the logging level [debug, info, warn, error, fatal, panic]",
			Value:  "debug",
			EnvVar: "LABD_LOG_LEVEL",
		},
		cli.StringFlag{
			Name:   "provider,p",
			Usage:  "set the provider to create node groups [inmemory, terraform]",
			Value:  "inmemory",
			EnvVar: "LABD_PROVIDER",
		},
		cli.StringFlag{
			Name:   "uploader,u",
			Usage:  "set the uploader to use to distribute p2p app binaries [file, s3]",
			Value:  "file",
			EnvVar: "LABD_UPLOADER",
		},
		cli.StringFlag{
			Name:   "uploader.s3.bucket",
			Usage:  "bucket name for s3 uploader",
			EnvVar: "LABD_UPLOADER_S3_BUCKET",
		},
		cli.StringFlag{
			Name:   "uploader.s3.prefix",
			Usage:  "bucket prefix for s3 uploader",
			EnvVar: "LABD_UPLOADER_S3_PREFIX",
		},
		cli.StringFlag{
			Name:   "uploader.s3.region",
			Usage:  "region for s3 uploader",
			EnvVar: "LABD_UPLOADER_S3_REGION",
		},
		cli.StringFlag{
			Name:   "uploader.file.address",
			Usage:  "address for file uploader",
			Value:  ":7000",
			EnvVar: "LABD_UPLOADER_FILE_ADDRESS",
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
	daemon, err := labd.New(root, c.GlobalString("address"), zerolog.Ctx(ctx),
		labd.WithLibp2pPort(c.GlobalInt("libp2p-port")),
		labd.WithProvider(c.GlobalString("provider")),
		labd.WithUploader(c.GlobalString("uploader")),
		labd.WithUploaderSettings(uploaders.UploaderSettings{
			S3: s3uploader.S3UploaderSettings{
				Bucket: c.GlobalString("uploader.s3.bucket"),
				Prefix: c.GlobalString("uploader.s3.prefix"),
				Region: c.GlobalString("uploader.s3.region"),
			},
			File: fileuploader.FileUploaderSettings{
				Address: c.GlobalString("uploader.file.address"),
			},
		}),
	)
	if err != nil {
		return err
	}
	defer daemon.Close()

	return daemon.Serve(ctx)
}
