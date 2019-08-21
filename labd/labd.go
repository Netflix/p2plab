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

package labd

import (
	"context"
	"io"
	"path/filepath"

	"github.com/Netflix/p2plab/daemon"
	"github.com/Netflix/p2plab/daemon/healthcheckrouter"
	"github.com/Netflix/p2plab/labd/routers/benchmarkrouter"
	"github.com/Netflix/p2plab/labd/routers/clusterrouter"
	"github.com/Netflix/p2plab/labd/routers/experimentrouter"
	"github.com/Netflix/p2plab/labd/routers/noderouter"
	"github.com/Netflix/p2plab/labd/routers/scenariorouter"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/peer"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/Netflix/p2plab/providers"
	"github.com/Netflix/p2plab/transformers"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type Labd struct {
	daemon  *daemon.Daemon
	seeder  *peer.Peer
	closers []io.Closer
}

func New(root, addr string, logger *zerolog.Logger) (*Labd, error) {
	var closers []io.Closer
	db, err := metadata.NewDB(root)
	if err != nil {
		return nil, err
	}
	closers = append(closers, db)

	client, err := httputil.NewClient(cleanhttp.DefaultClient(), httputil.WithLogger(logger))
	if err != nil {
		return nil, err
	}

	provider, err := providers.GetNodeProvider(filepath.Join(root, "providers"), "terraform")
	if err != nil {
		return nil, err
	}

	ts := transformers.New(filepath.Join(root, "transformers"))
	closers = append(closers, ts)

	sctx, cancel := context.WithCancel(context.Background())
	seeder, err := peer.New(sctx, filepath.Join(root, "seeder"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create seeder peer")
	}
	closers = append(closers, &daemon.CancelCloser{cancel})

	daemon := daemon.New(addr, logger,
		healthcheckrouter.New(),
		clusterrouter.New(db, provider, client),
		noderouter.New(db, client),
		scenariorouter.New(db),
		benchmarkrouter.New(db, client, ts, seeder),
		experimentrouter.New(db, provider, client, ts, seeder),
	)

	d := &Labd{
		daemon:  daemon,
		seeder:  seeder,
		closers: closers,
	}

	return d, nil
}

func (d *Labd) Close() error {
	for _, closer := range d.closers {
		err := closer.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Labd) Serve(ctx context.Context) error {
	var addrs []string
	for _, ma := range d.seeder.Host().Addrs() {
		addrs = append(addrs, ma.String())
	}
	zerolog.Ctx(ctx).Info().Strs("addrs", addrs).Msg("IPFS listening")

	return d.daemon.Serve(ctx)
}
