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
	// Create deploys a cluster.
	Create(ctx context.Context, name string, opts ...CreateClusterOption) error

	// Get returns a cluster.
	Get(ctx context.Context, name string) (Cluster, error)

	// Label adds/removes labels to/from clusters.
	Label(ctx context.Context, names, adds, removes []string) ([]Cluster, error)

	// List returns available clusters.
	List(ctx context.Context, opts ...ListOption) ([]Cluster, error)

	// Remove destroys clusters permanently.
	Remove(ctx context.Context, names ...string) error
}

// Cluster is a group of instances connected in a p2p network. They can be
// provisioned by developers, or CI. Clusters may span multiple regions and
// have heterogeneous nodes.
type Cluster interface {
	Labeled

	Metadata() metadata.Cluster

	// Update compiles a commit and updates the cluster to the new p2p
	// application.
	Update(ctx context.Context, commit string) error
}

// CreateClusterOption is an option to modify create cluster settings.
type CreateClusterOption func(*CreateClusterSettings) error

// CreateClusterSettings specify cluster properties for creation.
type CreateClusterSettings struct {
	Definition        string
	Size              int
	InstanceType      string
	Region            string
	ClusterDefinition metadata.ClusterDefinition
}

func WithClusterDefinition(definition string) CreateClusterOption {
	return func(s *CreateClusterSettings) error {
		s.Definition = definition
		return nil
	}
}

func WithClusterSize(size int) CreateClusterOption {
	return func(s *CreateClusterSettings) error {
		s.Size = size
		return nil
	}
}

func WithClusterInstanceType(instanceType string) CreateClusterOption {
	return func(s *CreateClusterSettings) error {
		s.InstanceType = instanceType
		return nil
	}
}

func WithClusterRegion(region string) CreateClusterOption {
	return func(s *CreateClusterSettings) error {
		s.Region = region
		return nil
	}
}

type ListOption func(*ListSettings) error

type ListSettings struct {
	Query string
}

func WithQuery(q string) ListOption {
	return func(s *ListSettings) error {
		s.Query = q
		return nil
	}
}

type QueryOption func(*QuerySettings) error

type QuerySettings struct {
	AddLabels    []string
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
