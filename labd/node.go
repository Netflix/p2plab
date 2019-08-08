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
	"github.com/Netflix/p2plab/labagent"
	"github.com/Netflix/p2plab/metadata"
)

type nodeAPI struct {
	cln *client
}

func (napi *nodeAPI) Get(ctx context.Context, cluster, id string) (p2plab.Node, error) {
	req := napi.cln.NewRequest("GET", "/clusters/%s/nodes/%s", cluster, id)
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var m metadata.Node
	err = json.NewDecoder(resp.Body).Decode(&m)
	if err != nil {
		return nil, err
	}

	return newNode(napi.cln, m), nil
}

type node struct {
	labdCln     *client
	labagentCln *labagent.Client
	metadata    metadata.Node
}

func newNode(cln *client, m metadata.Node) *node {
	return &node{
		labdCln:     cln,
		labagentCln: labagent.NewClient(m.Address),
		metadata:    m,
	}
}

func (n *node) Metadata() metadata.Node {
	return n.metadata
}

func (n *node) SSH(ctx context.Context, opts ...p2plab.SSHOption) error {
	return nil
}

func (n *node) Run(ctx context.Context, task metadata.Task) error {
	return n.labagentCln.Run(ctx, task)
}
