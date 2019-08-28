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

package httpdownloader

import (
	"context"
	"io"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/pkg/httputil"
)

type downloader struct {
	client *httputil.Client
}

func New(client *httputil.Client) p2plab.Downloader {
	return &downloader{client}
}

func (f *downloader) Download(ctx context.Context, link string) (io.ReadCloser, error) {
	req := f.client.NewRequest("GET", link)
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}
