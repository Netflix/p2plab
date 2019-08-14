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

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/errdefs"
	"github.com/Netflix/p2plab/metadata"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"github.com/rs/zerolog/log"
)

func Run(ctx context.Context, nset p2plab.NodeSet, plan metadata.ScenarioPlan, seederAddr string) error {
	log.Info().Msg("Seeding cluster")
	seed, gctx := errgroup.WithContext(ctx)
	for id, task := range plan.Seed {
		seed.Go(func() error {
			n := nset.Get(id)
			if n == nil {
				return errors.Wrapf(errdefs.ErrNotFound, "could not find node %q in node set", id)
			}

			err := n.Run(gctx, metadata.Task{
				Type:    metadata.TaskConnect,
				Subject: seederAddr,
			})
			if err != nil {
				return errors.Wrap(err, "failed to connect to seeding peer")
			}

			err = n.Run(gctx, task)
			if err != nil {
				return errors.Wrap(err, "failed to run seeding task")
			}

			err = n.Run(gctx, metadata.Task{
				Type:    metadata.TaskDisconnect,
				Subject: seederAddr,
			})
			if err != nil {
				return errors.Wrap(err, "failed to disconnect from seeding peer")
			}

			return nil
		})
	}
	err := seed.Wait()
	if err != nil {
		return err
	}

	log.Info().Msg("Benchmarking cluster")
	benchmark, gctx := errgroup.WithContext(ctx)
	for id, task := range plan.Benchmark {
		benchmark.Go(func() error {
			n := nset.Get(id)
			if n == nil {
				return errors.Wrapf(errdefs.ErrNotFound, "could not find node %q in node set", id)
			}

			return n.Run(gctx, task)
		})
	}
	err = benchmark.Wait()
	if err != nil {
		return err
	}

	return nil
}
