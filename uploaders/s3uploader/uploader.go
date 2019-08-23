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

package s3uploader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"time"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/pkg/logutil"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
	cid "github.com/ipfs/go-cid"
	multihash "github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type S3UploaderSettings struct {
	Bucket string
	Prefix string
	Region string
}

type uploader struct {
	bucket        string
	prefix        string
	uploadManager *s3manager.Uploader
	cidBuilder    cid.Builder
}

func New(client *http.Client, settings S3UploaderSettings) (p2plab.Uploader, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load aws config")
	}
	cfg.Region = settings.Region
	cfg.HTTPClient = client

	uploadManager := s3manager.NewUploader(cfg)
	return &uploader{
		bucket:        settings.Bucket,
		prefix:        settings.Prefix,
		uploadManager: uploadManager,
		cidBuilder:    cid.V1Builder{MhType: multihash.SHA2_256},
	}, nil
}

func (u *uploader) Upload(ctx context.Context, r io.Reader) (link string, err error) {
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}

	c, err := u.cidBuilder.Sum(content)
	if err != nil {
		return "", err
	}

	key := path.Join(u.prefix, c.String())
	logger := zerolog.Ctx(ctx).With().Str("bucket", u.bucket).Str("key", key).Logger()
	ectx, cancel := context.WithCancel(logger.WithContext(ctx))
	defer cancel()

	go logutil.Elapsed(ectx, 20*time.Second, "Uploading S3 object")

	upParams := &s3manager.UploadInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(content),
	}

	_, err = u.uploadManager.UploadWithContext(ctx, upParams)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("s3://%s/%s/%s", u.bucket, u.prefix, c), nil
}
