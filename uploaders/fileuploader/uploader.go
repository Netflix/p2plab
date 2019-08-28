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

package fileuploader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Netflix/p2plab"
	cid "github.com/ipfs/go-cid"
	multihash "github.com/multiformats/go-multihash"
	"github.com/rs/zerolog"
)

type FileUploaderSettings struct {
	Address string
}

type uploader struct {
	root       string
	cancel     context.CancelFunc
	cidBuilder cid.Builder
}

func New(root string, logger *zerolog.Logger, settings FileUploaderSettings) (p2plab.Uploader, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(root, 0711)
	if err != nil {
		return nil, err
	}

	s := &http.Server{
		Handler:           http.FileServer(http.Dir(root)),
		Addr:              settings.Address,
		ReadHeaderTimeout: 20 * time.Second,
		ReadTimeout:       1 * time.Minute,
		WriteTimeout:      30 * time.Minute,
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ctx.Done()
		err := s.Shutdown(ctx)
		if err != nil {
			logger.Error().Err(err).Msg("failed to shutdown fsuploader")
		}
	}()

	go func() {
		err := s.ListenAndServe()
		if err != nil {
			logger.Error().Err(err).Msg("failed to serve fsuploader")
		}
	}()

	return &uploader{
		root:       root,
		cancel:     cancel,
		cidBuilder: cid.V1Builder{MhType: multihash.SHA2_256},
	}, nil
}

func (u *uploader) Close() error {
	u.cancel()
	return nil
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

	link = fmt.Sprintf("file://%s/%s", u.root, c)
	uploadPath := filepath.Join(u.root, c.String())
	_, err = os.Stat(uploadPath)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	} else if err == nil {
		return link, nil
	}

	f, err := os.Create(uploadPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = io.Copy(f, bytes.NewReader(content))
	if err != nil {
		return "", err
	}

	return link, nil
}
