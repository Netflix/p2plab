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
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

func Session(ctx context.Context, ns []p2plab.Node, fn func(context.Context) error) error {
	eg, gctx := errgroup.WithContext(ctx)

	zerolog.Ctx(ctx).Info().Msg("Starting a session for benchmarking")
	go logutil.Elapsed(gctx, 20*time.Second, "Starting a session for benchmarking")

	cancels := make([]context.CancelFunc, len(ns))
	for i, n := range ns {
		i, n := i, n
		eg.Go(func() error {
			lctx, cancel := context.WithCancel(gctx)
			pdef := n.Metadata().Peer
			err := n.Update(lctx, "", pdef)
			if err != nil {
				return err
			}

			cancels[i] = cancel
			return nil
		})
	}

	err := eg.Wait()
	if err != nil {
		return err
	}

	err = fn(ctx)
	if err != nil {
		return err
	}

	zerolog.Ctx(ctx).Info().Msg("Ending the session")
	for _, cancel := range cancels {
		cancel()
	}

	return nil
}
