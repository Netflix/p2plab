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
}

// Node is an instance running the P2P application to be benchmarked.
type Node interface {
	Metadata() metadata.Node

	// SSH creates a SSH connection to the node.
	SSH(ctx context.Context, opts ...SSHOption) error

	// Run executes an task on the node and returns a channel that recieves
	// progress events on the completion of the task.
	Run(ctx context.Context, task metadata.Task) error
}

// NodeSet is a group of unique nodes.
type NodeSet interface {
	// Add adds a node to the set. If the node already exists in the set, it is
	// not added again.
	Add(node Node)

	// Remove removes a node from a set. If the node doesn't exist in the set,
	// it is not removed.
	Remove(node Node)

	Contains(node Node) bool

	Get(id string) Node

	// Slice returns a slice of nodes from the set.
	Slice() []Node
}

// SSHOption is an option to modify SSH settings.
type SSHOption func(SSHSettings) error

// SSHSetttings specify ssh settings when connecting to a node.
type SSHSettings struct {
}
