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
	"fmt"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/labagent"
	"github.com/Netflix/p2plab/labapp"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/httputil"
)

type nodeAPI struct {
	client *httputil.Client
	url    urlFunc
}

func (a *nodeAPI) Get(ctx context.Context, cluster, id string) (p2plab.Node, error) {
	req := a.client.NewRequest("GET", a.url("/clusters/%s/nodes/%s", cluster, id))
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

	return newNode(a.client, m), nil
}

type node struct {
	p2plab.AgentAPI
	p2plab.ApplicationAPI
	metadata metadata.Node
}

func newNode(client *httputil.Client, m metadata.Node) *node {
	return &node{
		AgentAPI:       labagent.NewControl(client, fmt.Sprintf("http://%s:7002", m.Address)),
		ApplicationAPI: labapp.NewControl(client, fmt.Sprintf("http://%s:7003", m.Address)),
		metadata:       m,
	}
}

func (n *node) Metadata() metadata.Node {
	return n.metadata
}
