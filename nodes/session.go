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

package nodes

import (
	"context"
	"time"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/pkg/logutil"
	"github.com/Netflix/p2plab/pkg/traceutil"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

func Session(ctx context.Context, ns []p2plab.Node, fn func(context.Context) error) (opentracing.Span, error) {
	span := traceutil.Tracer(ctx).StartSpan("scenarios.Session")
	defer span.Finish()
	sctx := opentracing.ContextWithSpan(ctx, span)

	var ids []string
	for _, n := range ns {
		ids = append(ids, n.ID())
	}

	eg, gctx := errgroup.WithContext(sctx)

	zerolog.Ctx(ctx).Info().Msg("Starting a session for benchmarking")
	go logutil.Elapsed(gctx, 20*time.Second, "Starting a session for benchmarking")
	/*var cancels = make([]context.CancelFunc, 0, len(ns))
	for _, n := range ns {
		n := n
		eg.Go(func() error {
			lctx, cancel := context.WithCancel(gctx)
			cancels = append(cancels, cancel)
			pdef := n.Metadata().Peer
			err := n.Update(lctx, n.ID(), "", pdef)
			if err != nil && !errdefs.IsCancelled(err) {
				return errors.Wrapf(err, "failed to update node %q", n.ID())
			}

			return nil
		})
	}
	*/
	err := WaitHealthy(ctx, ns)
	if err != nil {
		return nil, err
	}

	err = fn(sctx)
	if err != nil {
		return nil, err
	}

	/*zerolog.Ctx(ctx).Info().Strs("nodes", ids).Msg("Ending the session")
	for _, cancel := range cancels {
		cancel()
	}
	*/
	err = eg.Wait()
	if err != nil {
		return nil, err
	}

	return span, nil
}
