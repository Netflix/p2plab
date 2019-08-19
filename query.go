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

type Labeled interface {
	ID() string

	Labels() []string
}

type LabeledSet interface {
	Add(labeled Labeled)

	Remove(id string)

	Get(id string) Labeled

	Contains(id string) bool

	Slice() []Labeled
}

// Query is an executable function against a cluster to match a set of nodes.
// Queries are used to group nodes to perform actions in either the seeding or
// benchmarking stage of a scenario.
type Query interface {
	String() string

	Match(ctx context.Context, lset LabeledSet) (LabeledSet, error)
}
