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

import "context"

type ScenarioAPI interface {
	Create(ctx context.Context, name string, sdef ScenarioDefinition) (Scenario, error)

	Get(ctx context.Context, name string) (Scenario, error)

	List(ctx context.Context) ([]Scenario, error)
}

type Scenario interface {
	Remove(ctx context.Context) error
}

type ScenarioDefinition struct {
	Objects   map[string]ObjectDefinition `json:"objects,omitempty"`
	Seed      map[string]string           `json:"seed,omitempty"`
	Benchmark map[string]string           `json:"benchmark,omitempty"`
}

type ObjectDefinition struct {
	Type    string `json:"type"`
	Chunker string `json:"chunker"`
	Layout  string `json:"layout"`
}

