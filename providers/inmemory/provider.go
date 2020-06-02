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

package inmemory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/labagent"
	"github.com/Netflix/p2plab/metadata"
	"github.com/phayes/freeport"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
)

type provider struct {
	root      string
	nodes     map[string][]*node
	logger    *zerolog.Logger
	agentOpts []labagent.LabagentOption
	mu        sync.Mutex
}

func New(root string, db metadata.DB, logger *zerolog.Logger, agentOpts ...labagent.LabagentOption) (p2plab.NodeProvider, error) {
	err := os.MkdirAll(root, 0711)
	if err != nil {
		return nil, err
	}

	p := &provider{
		root:      root,
		nodes:     make(map[string][]*node),
		logger:    logger,
		agentOpts: agentOpts,
	}

	ctx := context.Background()
	clusters, err := db.ListClusters(ctx)
	if err != nil {
		return nil, err
	}

	for _, cluster := range clusters {
		nodes, err := db.ListNodes(ctx, cluster.ID)
		if err != nil {
			return nil, err
		}

		for _, node := range nodes {
			n, err := p.newNode(node.ID, node.AgentPort, node.AppPort)
			if err != nil {
				return nil, err
			}
			p.nodes[node.ID] = append(p.nodes[node.ID], n)
		}
	}

	return p, nil
}

func (p *provider) CreateNodeGroup(ctx context.Context, id string, cdef metadata.ClusterDefinition) (*p2plab.NodeGroup, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	numPorts := 0
	for _, group := range cdef.Groups {
		numPorts += group.Size
	}

	freePorts, err := freeport.GetFreePorts(numPorts * 2)
	if err != nil {
		return nil, err
	}

	var (
		ns        []metadata.Node
		portIndex = 0
	)
	for _, group := range cdef.Groups {
		for i := 0; i < group.Size; i++ {
			agentPort, appPort := freePorts[portIndex], freePorts[portIndex+1]
			portIndex += 2

			id := xid.New().String()
			n, err := p.newNode(id, agentPort, appPort)
			if err != nil {
				return nil, err
			}
			p.nodes[id] = append(p.nodes[id], n)

			ns = append(ns, metadata.Node{
				ID:        n.ID,
				Address:   "127.0.0.1",
				AgentPort: n.AgentPort,
				AppPort:   n.AppPort,
				Peer:      *group.Peer,
				Labels: append([]string{
					n.ID,
					group.InstanceType,
					group.Region,
				}, group.Labels...),
			})
		}
	}

	return &p2plab.NodeGroup{
		ID:    id,
		Nodes: ns,
	}, nil
}

func (p *provider) DestroyNodeGroup(ctx context.Context, ng *p2plab.NodeGroup) error {
	for _, n := range p.nodes[ng.ID] {
		err := n.Close()
		if err != nil {
			p.logger.Error().Err(err).Str("node.id", n.ID).Msg("error encountered while destroying node group")
			return err
		}
	}

	delete(p.nodes, ng.ID)
	return nil
}

type node struct {
	ID        string
	AgentPort int
	AppPort   int
	LabAgent  *labagent.LabAgent
	cancel    context.CancelFunc
}

func (p *provider) newNode(id string, agentPort, appPort int) (*node, error) {
	agentRoot := filepath.Join(p.root, id, "labagent")
	agentAddr := fmt.Sprintf(":%d", agentPort)
	err := os.MkdirAll(agentRoot, 0711)
	if err != nil {
		return nil, err
	}

	appRoot := filepath.Join(p.root, id, "labapp")
	appAddr := fmt.Sprintf("http://localhost:%d", appPort)
	err = os.MkdirAll(appRoot, 0711)
	if err != nil {
		return nil, err
	}

	la, err := labagent.New(agentRoot, agentAddr, appRoot, appAddr, p.logger, p.agentOpts...)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err := la.Serve(ctx)
		if err != nil {
			p.logger.Error().Err(err).Str("id", id).Msg("serve exited with error")
		}
	}()

	return &node{
		ID:        id,
		AgentPort: agentPort,
		AppPort:   appPort,
		LabAgent:  la,
		cancel:    cancel,
	}, nil
}

func (n *node) Close() error {
	n.cancel()
	return n.LabAgent.Close()
}
