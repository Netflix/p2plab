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

package scenarios

import (
	"context"
	"sync"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/actions"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/query"
	"github.com/Netflix/p2plab/transformers"
	"golang.org/x/sync/errgroup"
)

func Plan(ctx context.Context, peer p2plab.Peer, nset p2plab.NodeSet, sdef metadata.ScenarioDefinition) (metadata.ScenarioPlan, error) {
	plan := metadata.ScenarioPlan{
		Seed:      make(map[string]metadata.Task),
		Benchmark: make(map[string]metadata.Task),
	}

	objects, gctx := errgroup.WithContext(ctx)

	var mu sync.Mutex
	for name, odef := range sdef.Objects {
		objects.Go(func() error {
			t, err := transformers.GetTransformer(odef.Type)
			if err != nil {
				return err
			}

			c, err := t.Transform(gctx, peer, odef.Source, nil)
			if err != nil {
				return err
			}

			mu.Lock()
			plan.Objects[name] = c
			mu.Unlock()
			return nil
		})
	}

	for q, a := range sdef.Seed {
		qry, err := query.Parse(q)
		if err != nil {
			return plan, err
		}

		mset, err := qry.Match(ctx, nset)
		if err != nil {
			return plan, err
		}

		action, err := actions.Parse(a)
		if err != nil {
			return plan, err
		}

		taskMap, err := action.Tasks(ctx, mset)
		if err != nil {
			return plan, err
		}

		plan.Seed = taskMap
	}

	// TODO: Refactor `Seed` and `Benchmark` into arbitrary `Stages`.
	for q, a := range sdef.Benchmark {
		qry, err := query.Parse(q)
		if err != nil {
			return plan, err
		}

		mset, err := qry.Match(ctx, nset)
		if err != nil {
			return plan, err
		}

		action, err := actions.Parse(a)
		if err != nil {
			return plan, err
		}

		taskMap, err := action.Tasks(ctx, mset)
		if err != nil {
			return plan, err
		}

		plan.Benchmark = taskMap
	}

	return plan, nil
}
