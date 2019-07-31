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

// ClusterAPI defines API for cluster operations.
type ClusterAPI interface {
	// Create deploys a cluster of a p2p application.
	Create(ctx context.Context, id string, opts ...CreateClusterOption) (Cluster, error)

	// Get returns a cluster.
	Get(ctx context.Context, id string) (Cluster, error)

	// List returns available clusters.
	List(ctx context.Context) ([]Cluster, error)
}

// Cluster is a group of instances connected in a p2p network. They can be
// provisioned by developers, or CI. Clusters may span multiple regions and
// have heterogeneous nodes.
type Cluster interface {
	Metadata() metadata.Cluster

	// Remove destroys a cluster permanently.
	Remove(ctx context.Context) error

	// Query executes a query and returns a set of matching nodes.
	Query(ctx context.Context, q Query, opts ...QueryOption) (NodeSet, error)

	// Update compiles a commit and updates the cluster to the new p2p
	// application.
	Update(ctx context.Context, commit string) error
}

// CreateClusterOption is an option to modify create cluster settings.
type CreateClusterOption func(*CreateClusterSettings) error

// CreateClusterSettings specify cluster properties for creation.
type CreateClusterSettings struct {
	Size int
}

type QueryOption func(*QuerySettings) error

type QuerySettings struct {
	AddLabels []string
	RemoveLabels []string
}

func WithAddLabels(labels ...string) QueryOption {
	return func(s *QuerySettings) error {
		s.AddLabels = append(s.AddLabels, labels...)
		return nil
	}
}

func WithRemoveLabels(labels ...string) QueryOption {
	return func(s *QuerySettings) error {
		s.RemoveLabels = append(s.RemoveLabels, labels...)
		return nil
	}
}
