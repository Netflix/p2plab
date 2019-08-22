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
	"github.com/Netflix/p2plab/errdefs"
	"github.com/Netflix/p2plab/pkg/logutil"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

func WaitHealthy(ctx context.Context, ns []p2plab.Node) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cluster healthy")
	defer span.Finish()
	span.SetTag("nodes", len(ns))

	healthchecks, gctx := errgroup.WithContext(ctx)

	zerolog.Ctx(ctx).Info().Msg("Waiting for healthy nodes")
	go logutil.Elapsed(gctx, 20*time.Second, "Waiting for healthy nodes")
	for _, n := range ns {
		n := n
		healthchecks.Go(func() error {
			ok := n.Healthcheck(gctx)
			if !ok {
				return errors.Wrapf(errdefs.ErrUnavailable, "node %q", n.ID())
			}
			return nil
		})
	}

	err := healthchecks.Wait()
	if err != nil {
		return err
	}

	return nil

}
