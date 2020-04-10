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

package controlapi

import (
	"context"
	"encoding/json"
	"io"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/httputil"
)

type buildAPI struct {
	client *httputil.Client
	url    urlFunc
}

func (a *buildAPI) Get(ctx context.Context, id string) (p2plab.Build, error) {
	req := a.client.NewRequest("GET", a.url("/builds/%s/json", id))
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var m metadata.Build
	err = json.NewDecoder(resp.Body).Decode(&m)
	if err != nil {
		return nil, err
	}

	return NewBuild(a.client, a.url, m), nil
}

func (a *buildAPI) List(ctx context.Context) ([]p2plab.Build, error) {
	req := a.client.NewRequest("GET", a.url("/builds/json"))

	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var metadatas []metadata.Build
	err = json.NewDecoder(resp.Body).Decode(&metadatas)
	if err != nil {
		return nil, err
	}

	var ns []p2plab.Build
	for _, m := range metadatas {
		ns = append(ns, NewBuild(a.client, a.url, m))
	}

	return ns, nil
}

func (a *buildAPI) Upload(ctx context.Context, r io.Reader) (p2plab.Build, error) {
	req := a.client.NewRequest("POST", a.url("/builds/upload")).
		Body(r)

	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var m metadata.Build
	err = json.NewDecoder(resp.Body).Decode(&m)
	if err != nil {
		return nil, err
	}

	return NewBuild(a.client, a.url, m), nil
}

type build struct {
	client   *httputil.Client
	metadata metadata.Build
	url      urlFunc
}

func NewBuild(client *httputil.Client, url urlFunc, m metadata.Build) p2plab.Build {
	return &build{
		client:   client,
		url:      url,
		metadata: m,
	}
}

func (n *build) ID() string {
	return n.metadata.ID
}

func (n *build) Metadata() metadata.Build {
	return n.metadata
}

func (n *build) Open(ctx context.Context) (io.ReadCloser, error) {
	req := n.client.NewRequest("GET", n.url("/builds/%s/download", n.ID()))

	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}
