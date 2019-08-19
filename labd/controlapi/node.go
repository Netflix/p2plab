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
	"fmt"
	"strings"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/labagent/agentapi"
	"github.com/Netflix/p2plab/labapp/appapi"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/httputil"
)

type nodeAPI struct {
	client *httputil.Client
	url    urlFunc
}

func (a *nodeAPI) Get(ctx context.Context, cluster, id string) (p2plab.Node, error) {
	req := a.client.NewRequest("GET", a.url("/clusters/%s/nodes/%s/json", cluster, id))
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

	return NewNode(a.client, m), nil
}

func (a *nodeAPI) Label(ctx context.Context, cluster string, ids, adds, removes []string) ([]p2plab.Node, error) {
	req := a.client.NewRequest("PUT", a.url("/clusters/%s/nodes/label", cluster)).
		Option("ids", strings.Join(ids, ","))

	if len(adds) > 0 {
		req.Option("adds", strings.Join(adds, ","))
	}
	if len(removes) > 0 {
		req.Option("removes", strings.Join(removes, ","))
	}

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
		nodes = append(nodes, NewNode(a.client, m))
	}

	return nodes, nil
}

func (a *nodeAPI) List(ctx context.Context, cluster string, opts ...p2plab.ListOption) ([]p2plab.Node, error) {
	var settings p2plab.ListSettings
	for _, opt := range opts {
		err := opt(&settings)
		if err != nil {
			return nil, err
		}
	}

	req := a.client.NewRequest("GET", a.url("/clusters/%s/nodes/json", cluster))
	if settings.Query != "" {
		req.Option("query", settings.Query)
	}

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
		nodes = append(nodes, NewNode(a.client, m))
	}

	return nodes, nil
}

type node struct {
	p2plab.AgentAPI
	p2plab.AppAPI
	metadata metadata.Node
}

func NewNode(client *httputil.Client, m metadata.Node) p2plab.Node {
	return &node{
		AgentAPI: agentapi.New(client, fmt.Sprintf("http://%s:7002", m.Address)),
		AppAPI:   appapi.New(client, fmt.Sprintf("http://%s:7003", m.Address)),
		metadata: m,
	}
}

func (n *node) ID() string {
	return n.metadata.ID
}

func (n *node) Labels() []string {
	return n.metadata.Labels
}

func (n *node) Metadata() metadata.Node {
	return n.metadata
}
