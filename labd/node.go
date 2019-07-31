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
	"encoding/json"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
)

type nodeAPI struct {
	cln *client
}

func (napi *nodeAPI) Get(ctx context.Context, id string) (p2plab.Node, error) {
	req := napi.cln.NewRequest("GET", "/nodes/%s", id)
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	n := node{cln: napi.cln}
	err = json.NewDecoder(resp.Body).Decode(&n.metadata)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (napi *nodeAPI) List(ctx context.Context) ([]p2plab.Node, error) {
	req := napi.cln.NewRequest("GET", "/nodes")
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var metadatas []metadata.Node
	err = json.NewDecoder(resp.Body).Decode(&metadatas)
	if err != nil {
		return nil, err
	}

	var nodes []p2plab.Node
	for _, m := range metadatas {
		nodes = append(nodes, &node{cln: napi.cln, metadata: m})
	}

	return nodes, nil
}

type node struct {
	cln      *client
	metadata metadata.Node
}

func (n *node) Metadata() metadata.Node {
	return n.metadata
}

func (n *node) SSH(ctx context.Context, opts ...p2plab.SSHOption) error {
	return nil
}
