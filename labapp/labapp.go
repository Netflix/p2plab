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

package labapp

import (
	"context"
	"io"

	"github.com/Netflix/p2plab/daemon"
	"github.com/Netflix/p2plab/labapp/approuter"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/peer"
	"github.com/rs/zerolog"
)

type LabApp struct {
	daemon  *daemon.Daemon
	peer    *peer.Peer
	closers []io.Closer
}

func New(root, addr string, port int, logger *zerolog.Logger, pdef metadata.PeerDefinition) (*LabApp, error) {
	var closers []io.Closer
	pctx, cancel := context.WithCancel(context.Background())
	p, err := peer.New(pctx, root, port, pdef)
	if err != nil {
		return nil, err
	}
	closers = append(closers, &daemon.CancelCloser{cancel})

	daemon, err := daemon.New("labapp", addr, logger,
		approuter.New(p),
	)
	if err != nil {
		return nil, err
	}
	closers = append(closers, daemon)

	return &LabApp{
		daemon:  daemon,
		peer:    p,
		closers: closers,
	}, nil
}

func (a *LabApp) Close() error {
	for _, closer := range a.closers {
		err := closer.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *LabApp) Serve(ctx context.Context) error {
	var addrs []string
	for _, ma := range a.peer.Host().Addrs() {
		addrs = append(addrs, ma.String())
	}
	zerolog.Ctx(ctx).Info().Msgf("IPFS listening on %s", addrs)

	return a.daemon.Serve(ctx)
}
