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

package labd

import (
	"context"

	"github.com/Netflix/p2plab"
)

type clusterAPI struct {
	cln *client
}

func (capi *clusterAPI) Create(ctx context.Context, opts ...p2plab.CreateClusterOption) (p2plab.Cluster, error) {
	settings := p2plab.CreateClusterSettings{
		Size: 3,
	}
	for _, opt := range opts {
		err := opt(settings)
		if err != nil {
			return nil, err
		}
	}

	req := capi.cln.NewRequest("POST", "/clusters").
		Option("size", settings.Size).
		Body("hello")


	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &cluster{
		cln: capi.cln,
	}, nil
}

func (capi *clusterAPI) Get(ctx context.Context, id string) (p2plab.Cluster, error) {
	req := capi.cln.NewRequest("HEAD", "/clusters/%s", id)
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &cluster{
		cln: capi.cln,
		id:  id,
	}, nil
}

func (capi *clusterAPI) List(ctx context.Context) ([]p2plab.Cluster, error) {
	req := capi.cln.NewRequest("GET", "/clusters")
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return nil, nil
}

type cluster struct {
	cln *client
	id  string
}

func (c *cluster) Remove(ctx context.Context) error {
	req := c.cln.NewRequest("DELETE", "/clusters/%s", c.id)
	resp, err := req.Send(ctx)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *cluster) Query(ctx context.Context, q p2plab.Query) (p2plab.NodeSet, error) {
	req := c.cln.NewRequest("POST", "/clusters/%s/query", c.id)
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return nil, nil
}

func (c *cluster) Update(ctx context.Context, commit string) error {
	req := c.cln.NewRequest("PUT", "/clusters/%s", c.id)
	resp, err := req.Send(ctx)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
