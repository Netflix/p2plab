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
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

func Run(ctx context.Context, lset p2plab.LabeledSet, plan metadata.ScenarioPlan, seederAddr string) error {
	zerolog.Ctx(ctx).Info().Msg("Seeding cluster")
	seed, gctx := errgroup.WithContext(ctx)
	for id, task := range plan.Seed {
		id, task := id, task
		seed.Go(func() error {
			labeled := lset.Get(id)
			if labeled == nil {
				return errors.Wrapf(errdefs.ErrNotFound, "could not find %q in labeled set", id)
			}
			logger := zerolog.Ctx(ctx).With().Str("node", id).Logger()

			n, ok := labeled.(p2plab.Node)
			if !ok {
				return errors.Wrap(errdefs.ErrInvalidArgument, "could not cast labeled to node")
			}

			logger.Debug().Str("addr", seederAddr).Msg("Connecting to seeding peer")
			err := n.Run(gctx, metadata.Task{
				Type:    metadata.TaskConnect,
				Subject: seederAddr,
			})
			if err != nil {
				return errors.Wrap(err, "failed to connect to seeding peer")
			}

			logger.Debug().Str("task", string(task.Type)).Msg("Executing seeding task")
			err = n.Run(gctx, task)
			if err != nil {
				return errors.Wrap(err, "failed to run seeding task")
			}

			logger.Debug().Str("addr", seederAddr).Msg("Disconnecting from seeding peer")
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

	zerolog.Ctx(ctx).Info().Msg("Benchmarking cluster")
	benchmark, gctx := errgroup.WithContext(ctx)
	for id, task := range plan.Benchmark {
		id, task := id, task
		benchmark.Go(func() error {
			labeled := lset.Get(id)
			if labeled == nil {
				return errors.Wrapf(errdefs.ErrNotFound, "could not find %q in labeled set", id)
			}
			logger := zerolog.Ctx(ctx).With().Str("node", id).Logger()

			n, ok := labeled.(p2plab.Node)
			if !ok {
				return errors.Wrap(errdefs.ErrInvalidArgument, "could not cast labeled to node")
			}

			logger.Debug().Str("task", string(task.Type)).Msg("Executing benchmarking task")
			return n.Run(gctx, task)
		})
	}

	err = benchmark.Wait()
	if err != nil {
		return err
	}

	zerolog.Ctx(ctx).Info().Msg("Benchmark completed")
	return nil
}
