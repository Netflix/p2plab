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
	"os"

	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/Netflix/p2plab/uploaders/s3uploader"
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
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "s3upload: must specify path to binary to upload")
		os.Exit(1)
	}

	err := run(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "s3upload: %s\n", err)
		os.Exit(1)
	}
}

func run(filename string) error {
	client := httputil.NewHTTPClient()
	uploader, err := s3uploader.New(client, s3uploader.S3UploaderSettings{
		Bucket: os.Getenv("S3_UPLOADER_BUCKET"),
		Prefix: os.Getenv("S3_UPLOADER_PREFIX"),
	})
	if err != nil {
		return err
	}

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	ctx := logger.WithContext(context.Background())
	ref, err := uploader.Upload(ctx, f)
	if err != nil {
		return err
	}

	zerolog.Ctx(ctx).Info().Str("ref", ref).Str("path", filename).Msg("Completed upload")
	return nil
}
