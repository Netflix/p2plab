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
	"fmt"
	"strings"
	"time"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/logutil"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

func Connect(ctx context.Context, ns []p2plab.Node) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cluster connect")
	defer span.Finish()
	span.SetTag("nodes", len(ns))

	collectPeerAddrs, gctx := errgroup.WithContext(ctx)

	go logutil.Elapsed(gctx, 20*time.Second, func(ctx context.Context, elapsed time.Duration) {
		zerolog.Ctx(ctx).Info().Dur("elapsed", elapsed).Msg("Retrieving peer infos")
	})

	peerAddrs := make([]string, len(ns))
	zerolog.Ctx(ctx).Info().Msg("Retrieving peer infos")
	for i, n := range ns {
		i, n := i, n
		collectPeerAddrs.Go(func() error {
			peerInfo, err := n.PeerInfo(gctx)
			if err != nil {
				return err
			}

			if len(peerInfo.Addrs) == 0 {
				return errors.Errorf("peer %q has zero addresses", n.Metadata().Address)
			}

			peerAddrs[i] = fmt.Sprintf("/ip4/%s/tcp/4001/p2p/%s", n.Metadata().Address, peerInfo.ID)
			return nil
		})
	}

	err := collectPeerAddrs.Wait()
	if err != nil {
		return err
	}

	connectPeers, gctx := errgroup.WithContext(ctx)

	go logutil.Elapsed(gctx, 20*time.Second, func(ctx context.Context, elapsed time.Duration) {
		zerolog.Ctx(ctx).Info().Dur("elapsed", elapsed).Msg("Connecting cluster")
	})

	zerolog.Ctx(ctx).Info().Msg("Connecting cluster")
	for _, n := range ns {
		n := n
		connectPeers.Go(func() error {
			return n.Run(gctx, metadata.Task{
				Type:    metadata.TaskConnect,
				Subject: strings.Join(peerAddrs, ","),
			})
		})
	}

	err = connectPeers.Wait()
	if err != nil {
		return err
	}

	return nil
}
