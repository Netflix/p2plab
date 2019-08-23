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

package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/Netflix/p2plab/downloaders/s3downloader"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	// UNIX Time is faster and smaller than most timestamps. If you set
	// zerolog.TimeFieldFormat to an empty string, logs will write with UNIX
	// time.
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "s3download: must specify ref and path to save file")
		os.Exit(1)
	}

	err := run(os.Args[1], os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "s3download: %s\n", err)
		os.Exit(1)
	}
}

func run(ref, filename string) error {
	client := httputil.NewHTTPClient()
	downloader, err := s3downloader.New(client, s3downloader.S3DownloaderSettings{
		Region: os.Getenv("LABAGENT_DOWNLOADER_S3_REGION"),
	})
	if err != nil {
		return err
	}

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	ctx := logger.WithContext(context.Background())

	rc, err := downloader.Download(ctx, ref)
	if err != nil {
		return err
	}
	defer rc.Close()

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := io.Copy(f, rc)
	if err != nil {
		return err
	}

	zerolog.Ctx(ctx).Info().Str("ref", ref).Str("path", filename).Int64("bytes", n).Msg("Completed download")
	return nil
}
