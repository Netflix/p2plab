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

package s3downloader

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/errdefs"
	"github.com/Netflix/p2plab/pkg/logutil"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type downloader struct {
	downloadManager *s3manager.Downloader
}

func New(client *http.Client) (p2plab.Downloader, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load aws config")
	}
	cfg.Region = endpoints.UsWest2RegionID
	cfg.HTTPClient = client

	return &downloader{
		downloadManager: s3manager.NewDownloader(cfg),
	}, nil
}

func (f *downloader) Download(ctx context.Context, link string) (io.ReadCloser, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, errors.Wrapf(errdefs.ErrInvalidArgument, "invalid url %q", link)
	}
	bucket := u.Host

	if len(u.Path) == 0 {
		return nil, errors.Wrap(errdefs.ErrInvalidArgument, "zero length s3 key")
	}
	key := u.Path[1:]

	logger := zerolog.Ctx(ctx).With().Str("bucket", bucket).Str("key", key).Logger()
	ectx, cancel := context.WithCancel(logger.WithContext(ctx))
	defer cancel()

	go logutil.Elapsed(ectx, 20*time.Second, "Downloading S3 object")

	buf := aws.NewWriteAtBuffer([]byte{})
	n, err := f.downloadManager.DownloadWithContext(ctx, buf, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	zerolog.Ctx(ctx).Debug().Int64("bytes", n).Msg("Downloaded object from S3")
	return ioutil.NopCloser(bytes.NewReader(buf.Bytes())), nil
}
