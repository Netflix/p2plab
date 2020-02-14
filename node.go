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

package p2plab

import (
	"context"

	"github.com/Netflix/p2plab/metadata"
)

// NodeAPI defines the API for node operations.
type NodeAPI interface {
	// Get returns a node.
	Get(ctx context.Context, cluster, id string) (Node, error)

	Label(ctx context.Context, cluster string, ids, adds, removes []string) ([]Node, error)

	List(ctx context.Context, cluster string, opts ...ListOption) ([]Node, error)
}

// Node is an instance running the P2P application to be benchmarked.
type Node interface {
	Labeled

	AgentAPI

	AppAPI

	Metadata() metadata.Node
}

// NodeProvider is a service that can provision nodes.
type NodeProvider interface {
	// CreateNodeGroup returns a healthy cluster of nodes.
	CreateNodeGroup(ctx context.Context, id string, cdef metadata.ClusterDefinition) (*NodeGroup, error)

	// DestroyNodeGroup destroys a cluster of nodes.
	DestroyNodeGroup(ctx context.Context, ng *NodeGroup) error
}

// NodeGroup is a cluster of nodes.
type NodeGroup struct {
	ID    string
	Nodes []metadata.Node
}

// SSHOption is an option to modify SSH settings.
type SSHOption func(SSHSettings) error

// SSHSetttings specify ssh settings when connecting to a node.
type SSHSettings struct {
}
