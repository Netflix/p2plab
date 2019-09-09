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

func Reset(ctx context.Context, ns []p2plab.Node) error {
	resetPeers, gctx := errgroup.WithContext(ctx)

	zerolog.Ctx(ctx).Info().Msg("Resetting cluster")
	go logutil.Elapsed(gctx, 20*time.Second, "Resetting cluster")

	for _, n := range ns {
		n := n
		resetPeers.Go(func() error {
			pdef := n.Metadata().Peer
			return n.Update(gctx, "", pdef)
		})
	}

	err := resetPeers.Wait()
	if err != nil {
		return err
	}

	return nil
}
