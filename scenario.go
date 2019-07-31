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

// ScenarioAPI defines API for scenario operations.
type ScenarioAPI interface {
	// Create saves a scenario for the given scenario definition.
	Create(ctx context.Context, name string, sdef ScenarioDefinition) (Scenario, error)

	// Get returns a scenario.
	Get(ctx context.Context, name string) (Scenario, error)

	// List returns available scenarios.
	List(ctx context.Context) ([]Scenario, error)
}

// Scenario is a schema for benchmarks that describes objects to benchmark, how
// the cluster is initially seeded, and what to benchmark.
type Scenario interface {
	Metadata() metadata.Scenario

	// Remove deletes a scenario.
	Remove(ctx context.Context) error
}

// ScenarioDefinition defines a scenario.
type ScenarioDefinition struct {
	// Objects map a unique object name to its definition. The definition
	// describes data that will be distributed during the benchmark. All objects
	// are initialized in parallel.
	Objects map[string]ObjectDefinition `json:"objects,omitempty"`

	// Seed map a query to an action. Queries are executed in parallel to seed
	// a cluster with initial data before running the benchmark.
	Seed map[string]string `json:"seed,omitempty"`

	// Benchmark maps a query to an action. Queries are executed in parallel
	// during the benchmark and metrics are collected during this stage.
	Benchmark map[string]string `json:"benchmark,omitempty"`
}

// ObjectDefinition define a type of data that will be distributed during the
// benchmark. The definition also specify options on how the data is converted
// into IPFS datastructures.
type ObjectDefinition struct {
	// Type specifies what type is the source of the data and how the data is
	// retrieved. Types must be one of the following: ["oci-image"].
	Type string `json:"type"`

	// Chunker specify which chunking algorithm to use to chunk the data into IPLD
	// blocks.
	Chunker string `json:"chunker"`

	// Layout specify how the DAG is shaped and constructed over the IPLD blocks.
	Layout string `json:"layout"`
}

// ObjectType is the type of data retrieved.
type ObjectType string

var (
	// ObjectContainerImage indicates that the object is an OCI image.
	ObjectContainerImage ObjectType = "oci-image"
)
