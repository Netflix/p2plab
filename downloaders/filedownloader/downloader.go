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

package filedownloader

import (
	"context"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/errdefs"
	"github.com/pkg/errors"
)

type downloader struct {
}

func New() p2plab.Downloader {
	return &downloader{}
}

func (f *downloader) Download(ctx context.Context, link string) (io.ReadCloser, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, errors.Wrapf(errdefs.ErrInvalidArgument, "invalid url %q", link)
	}

	downloadPath := filepath.Join(u.Host, u.Path)
	return os.Open(downloadPath)
}
