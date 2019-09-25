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
	"github.com/Netflix/p2plab/pkg/traceutil"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
)

func Connect(ctx context.Context, ns []p2plab.Node) error {
	span, ctx := traceutil.StartSpanFromContext(ctx, "nodes.Connect")
	defer span.Finish()
	span.SetTag("nodes", len(ns))

	collectPeerAddrs, gctx := errgroup.WithContext(ctx)

	zerolog.Ctx(ctx).Info().Msg("Retrieving peer infos")
	go logutil.Elapsed(gctx, 20*time.Second, "Retrieving peer infos")

	var peerInfos []peerstore.PeerInfo
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

			peerInfos = append(peerInfos, peerInfo)

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
	conns := make(map[peer.ID][]string)
	for _, n := range ns {
		npi, err := n.PeerInfo(ctx)
		if err != nil {
			return err
		}

		var peerAddrs []string
		for _, pi := range peerInfos {
			if _, ok := conns[pi.ID]; !ok && pi.ID != npi.ID {
				// for _, ma := range pi.Addrs {
				// 	peerAddrs = append(peerAddrs, fmt.Sprintf("%s/p2p/%s", ma, pi.ID))
				// }
				// TODO: for now only adding the node's first address, to avoid
				// disconnects that appear to occur when we dial more than one address
				// at the same time
				peerAddrs = append(peerAddrs, fmt.Sprintf("%s/p2p/%s", pi.Addrs[0], pi.ID))
			}
		}
		if len(peerAddrs) > 0 {
			conns[npi.ID] = peerAddrs
		}
	}

	connectPeers, gctx := errgroup.WithContext(ctx)

	zerolog.Ctx(ctx).Info().Msg("Connecting cluanster")
	go logutil.Elapsed(gctx, 20*time.Second, "Connecting cluster")

	for _, n := range ns {
		n := n
		npi, err := n.PeerInfo(gctx)
		if err != nil {
			return err
		}

		peerAddrs, ok := conns[npi.ID]
		if ok && len(peerAddrs) > 0 {
			// fmt.Print(npi.ID)
			// fmt.Println(":")
			// for _, pa := range peerAddrs {
			// 	fmt.Print("  ")
			// 	fmt.Println(pa)
			// }
			connectPeers.Go(func() error {
				return n.Run(gctx, metadata.Task{
					Type:    metadata.TaskConnect,
					Subject: strings.Join(peerAddrs, ","),
				})
			})
		}
	}

	err = connectPeers.Wait()
	if err != nil {
		return err
	}

	return nil
}
