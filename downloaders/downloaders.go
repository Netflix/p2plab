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

package downloaders

import (
	"sync"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/downloaders/httpdownloader"
	"github.com/Netflix/p2plab/downloaders/s3downloader"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/pkg/errors"
)

type DownloaderSettings struct {
	Client *httputil.Client
}

type Downloaders struct {
	root   string
	client *httputil.Client
	mu     sync.Mutex
	fs     map[string]p2plab.Downloader
}

func New(root string, settings DownloaderSettings) *Downloaders {
	return &Downloaders{
		root:   root,
		client: settings.Client,
		fs:     make(map[string]p2plab.Downloader),
	}
}

func (f *Downloaders) Get(downloaderType string) (p2plab.Downloader, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	downloader, ok := f.fs[downloaderType]
	if !ok {
		var err error
		downloader, err = f.newDownloader(downloaderType)
		if err != nil {
			return nil, err
		}
		f.fs[downloaderType] = downloader
	}
	return downloader, nil
}

func (f *Downloaders) newDownloader(downloaderType string) (p2plab.Downloader, error) {
	// root := filepath.Join(f.root, downloaderType)
	switch downloaderType {
	case "s3":
		return s3downloader.New(f.client.HTTPClient)
	case "http", "https":
		return httpdownloader.New(f.client), nil
	default:
		return nil, errors.Errorf("unrecognized downloader type: %q", downloaderType)
	}

}
