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

	cid "github.com/ipfs/go-cid"
)

// Labd defines APIs for labd.
type LabdAPI interface {
	// Cluster returns an implementaiton of Cluster API.
	Cluster() ClusterAPI

	// Node returns an implementation of Node API.
	Node() NodeAPI

	// Scenario returns an implementation of Scenario API.
	Scenario() ScenarioAPI

	// Benchmark returns an implementation of Benchmark API.
	Benchmark() BenchmarkAPI
}

// LabAgentAPI defines APIs for labagent.
type LabAgentAPI interface {
	// Get returns a reader to a file as identified as by its root CID. The file
	// must have been added as a UnixFS DAG.
	Get(ctx context.Context, target cid.Cid) error

	// Update fetches a new binary from the url and restarts its P2P application
	// with it.
	Update(ctx context.Context, url string) error
}
