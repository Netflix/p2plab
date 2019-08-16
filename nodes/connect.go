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
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

func WaitHealthy(ctx context.Context, nset p2plab.NodeSet) error {
	ns := nset.Slice()

	healthchecks, gctx := errgroup.WithContext(ctx)

	var cancel context.CancelFunc
	gctx, cancel = context.WithTimeout(gctx, 5*time.Minute)
	defer cancel()

	for _, n := range ns {
		n := n
		healthchecks.Go(func() error {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			healthCount := 0
			healthyThreshold := 5
			for {
				select {
				case <-gctx.Done():
					return errors.Errorf("timed out waiting for node %q to be healthy", n.Metadata().ID)
				case <-ticker.C:
					ok := n.Healthcheck(gctx)
					if ok {
						healthCount++
					}

					if healthCount == healthyThreshold {
						return nil
					}
				}
			}
		})
	}

	err := healthchecks.Wait()
	if err != nil {
		return err
	}

	return nil

}

func Connect(ctx context.Context, nset p2plab.NodeSet) error {
	err := WaitHealthy(ctx, nset)
	if err != nil {
		return err
	}

	ns := nset.Slice()
	peerAddrs := make([]string, len(ns))
	collectPeerAddrs, gctx := errgroup.WithContext(ctx)
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

	err = collectPeerAddrs.Wait()
	if err != nil {
		return err
	}

	connectPeers, gctx := errgroup.WithContext(ctx)
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
