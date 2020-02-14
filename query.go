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

// Labeled defines a resource that has labels.
type Labeled interface {
	// ID returns a uniquely identifiable string.
	ID() string

	// Labels returns a unique list of labels.
	Labels() []string
}

// LabeledSet is a set of labeled resources, duplicate resources are detected
// by the ID of the labeled resource.
type LabeledSet interface {
	// Add adds a labeled resource to the set.
	Add(labeled Labeled)

	// Remove removes a labeled resource from the set.
	Remove(id string)

	// Get returns a labeled resource from the set.
	Get(id string) Labeled

	// Contains returns whether a labeled resource with the id exists in the set.
	Contains(id string) bool

	// Slice returns the labeled resources as a slice.
	Slice() []Labeled
}

// Query is an executable function against a cluster to match a set of nodes.
// Queries are used to group nodes to perform actions in either the seeding or
// benchmarking stage of a scenario.
type Query interface {
	// String returns the original query.
	String() string

	// Match returns the subset of lset that matches the query.
	Match(ctx context.Context, lset LabeledSet) (LabeledSet, error)
}
