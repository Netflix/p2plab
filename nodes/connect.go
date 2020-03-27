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
	"sync"
	"time"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/logutil"
	"github.com/Netflix/p2plab/pkg/traceutil"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

func Connect(ctx context.Context, ns []p2plab.Node) error {
	span, ctx := traceutil.StartSpanFromContext(ctx, "nodes.Connect")
	defer span.Finish()
	span.SetTag("nodes", len(ns))

	collectPeerAddrs, gctx := errgroup.WithContext(ctx)

	zerolog.Ctx(ctx).Info().Msg("Retrieving peer infos")
	go logutil.Elapsed(gctx, 20*time.Second, "Retrieving peer infos")

	var lk sync.Mutex
	peerInfoByNodeID := make(map[string]peer.AddrInfo)
	for _, n := range ns {
		n := n
		collectPeerAddrs.Go(func() error {
			peerInfo, err := n.PeerInfo(gctx)
			if err != nil {
				return err
			}

			if len(peerInfo.Addrs) == 0 {
				return errors.Errorf("peer %q has zero addresses", n.Metadata().Address)
			}

			lk.Lock()
			peerInfoByNodeID[n.ID()] = peerInfo
			lk.Unlock()

			for _, ma := range peerInfo.Addrs {
				zerolog.Ctx(gctx).Debug().Str("addr", ma.String()).Msg("Retrieved peer address")
			}

			return nil
		})
	}

	err := collectPeerAddrs.Wait()
	if err != nil {
		return err
	}

	// Work out which addresses each node should dial, such that all dials
	// are in one direction (nodes don't dial a node that dialed them).
	// This helps avoid a libp2p bug where nodes that simultaneously dial
	// each other can get random disconnects.
	conns := make(map[peer.ID]map[peer.ID][]string)
	for _, n := range ns {
		npi, ok := peerInfoByNodeID[n.ID()]
		if !ok {
			panic("Should have peer info for every node")
		}

		conns[npi.ID] = make(map[peer.ID][]string)
		for _, pi := range peerInfoByNodeID {
			_, ok := conns[pi.ID]
			if ok || pi.ID == npi.ID {
				continue
			}

			var peerAddrs []string
			for _, ma := range pi.Addrs {
				peerAddrs = append(peerAddrs, fmt.Sprintf("%s/p2p/%s", ma, pi.ID))
			}
			if len(peerAddrs) > 0 {
				conns[npi.ID][pi.ID] = peerAddrs
			}
		}
	}

	connectPeers, gctx := errgroup.WithContext(ctx)

	zerolog.Ctx(ctx).Info().Msg("Connecting cluster")
	go logutil.Elapsed(gctx, 20*time.Second, "Connecting cluster")

	for _, n := range ns {
		npi, ok := peerInfoByNodeID[n.ID()]
		if !ok {
			panic("Should have peer info for every node")
		}

		toConns, ok := conns[npi.ID]
		if !ok {
			continue
		}

		for _, peerAddrs := range toConns {
			if len(peerAddrs) == 0 {
				continue
			}

			n := n
			peerAddrs := peerAddrs
			connectPeers.Go(func() error {
				return n.Run(ctx, metadata.Task{
					Type:    metadata.TaskConnectOne,
					Subject: strings.Join(peerAddrs, ","),
				})
			})
		}
	}

	return connectPeers.Wait()
}
