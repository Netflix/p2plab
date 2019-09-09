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
	"strings"
	"time"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/errdefs"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/nodes"
	"github.com/Netflix/p2plab/pkg/logutil"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type Execution struct {
	Start time.Time
	End   time.Time
	Span  opentracing.Span
}

func Run(ctx context.Context, lset p2plab.LabeledSet, plan metadata.ScenarioPlan, seederAddrs []string) (*Execution, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "scenario run")
	defer span.Finish()

	err := Seed(ctx, lset, plan.Seed, seederAddrs)
	if err != nil {
		return nil, err
	}

	ns, err := LabeledSetToNodes(lset)
	if err != nil {
		return nil, err
	}

	err = nodes.Reset(ctx, ns)
	if err != nil {
		return nil, err
	}

	err = nodes.Connect(ctx, ns)
	if err != nil {
		return nil, err
	}

	time.Sleep(time.Second)

	start := time.Now()
	benchmarkSpan, err := Benchmark(ctx, lset, plan.Benchmark)
	if err != nil {
		return nil, err
	}
	end := time.Now()

	return &Execution{
		Start: start,
		End:   end,
		Span:  benchmarkSpan,
	}, nil
}

func LabeledSetToNodes(lset p2plab.LabeledSet) ([]p2plab.Node, error) {
	var ns []p2plab.Node
	for _, l := range lset.Slice() {
		n, ok := l.(p2plab.Node)
		if !ok {
			return nil, errors.Wrap(errdefs.ErrInvalidArgument, "lset contains elements not p2plab.Node")
		}
		ns = append(ns, n)
	}
	return ns, nil
}

func Seed(ctx context.Context, lset p2plab.LabeledSet, seed metadata.ScenarioStage, seederAddrs []string) error {
	seeding, gctx := errgroup.WithContext(ctx)

	zerolog.Ctx(ctx).Info().Msg("Seeding cluster")
	go logutil.Elapsed(gctx, 20*time.Second, "Seeding cluster")
	for id, task := range seed {
		id, task := id, task
		seeding.Go(func() error {
			labeled := lset.Get(id)
			if labeled == nil {
				return errors.Wrapf(errdefs.ErrNotFound, "could not find %q in labeled set", id)
			}
			logger := zerolog.Ctx(ctx).With().Str("node", id).Logger()

			n, ok := labeled.(p2plab.Node)
			if !ok {
				return errors.Wrap(errdefs.ErrInvalidArgument, "could not cast labeled to node")
			}

			logger.Debug().Strs("addrs", seederAddrs).Msg("Connecting to seeding peer")
			err := n.Run(gctx, metadata.Task{
				Type:    metadata.TaskConnect,
				Subject: strings.Join(seederAddrs, ","),
			})
			if err != nil {
				return errors.Wrap(err, "failed to connect to seeding peer")
			}

			logger.Debug().Str("task", string(task.Type)).Msg("Executing seeding task")
			err = n.Run(gctx, task)
			if err != nil {
				return errors.Wrap(err, "failed to run seeding task")
			}

			logger.Debug().Strs("addrs", seederAddrs).Msg("Disconnecting from seeding peer")
			err = n.Run(gctx, metadata.Task{
				Type:    metadata.TaskDisconnect,
				Subject: strings.Join(seederAddrs, ","),
			})
			if err != nil {
				return errors.Wrap(err, "failed to disconnect from seeding peer")
			}

			return nil
		})
	}

	err := seeding.Wait()
	if err != nil {
		return err
	}

	zerolog.Ctx(ctx).Info().Msg("Seeding completed")
	return nil
}

func Benchmark(ctx context.Context, lset p2plab.LabeledSet, benchmark metadata.ScenarioStage) (opentracing.Span, error) {
	span := opentracing.StartSpan("cluster benchmark")
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	benchmarking, gctx := errgroup.WithContext(ctx)

	zerolog.Ctx(ctx).Info().Msg("Benchmarking cluster")
	go logutil.Elapsed(gctx, 20*time.Second, "Benchmarking cluster")
	for id, task := range benchmark {
		id, task := id, task
		benchmarking.Go(func() error {
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

	err := benchmarking.Wait()
	if err != nil {
		return nil, err
	}

	zerolog.Ctx(ctx).Info().Msg("Benchmark completed")
	return span, nil
}
