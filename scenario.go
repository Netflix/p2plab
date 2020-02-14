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
	Create(ctx context.Context, name string, sdef metadata.ScenarioDefinition) (Scenario, error)

	// Get returns a scenario.
	Get(ctx context.Context, name string) (Scenario, error)

	// Label adds and removes labels from nodes identified by the list of names.
	Label(ctx context.Context, names, adds, removes []string) ([]Scenario, error)

	// List returns available scenarios.
	List(ctx context.Context, opts ...ListOption) ([]Scenario, error)

	Remove(ctx context.Context, names ...string) error
}

// Scenario is a schema for benchmarks that describes objects to benchmark, how
// the cluster is initially seeded, and what to benchmark.
type Scenario interface {
	Labeled

	Metadata() metadata.Scenario
}
