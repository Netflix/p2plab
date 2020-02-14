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

// ExperimentAPI is an unimplemented layer to run experiments, a collection
// of benchmarks while varying some aspect.
type ExperimentAPI interface {
	Create(ctx context.Context, id string, edef metadata.ExperimentDefinition) (Experiment, error)

	Get(ctx context.Context, id string) (Experiment, error)

	Label(ctx context.Context, ids, adds, removes []string) ([]Experiment, error)

	List(ctx context.Context, opts ...ListOption) ([]Experiment, error)

	Remove(ctx context.Context, ids ...string) error
}

type Experiment interface {
	Labeled

	Metadata() metadata.Experiment
}
