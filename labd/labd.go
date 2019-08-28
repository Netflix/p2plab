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

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/builder"
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
	"github.com/Netflix/p2plab/uploaders"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type Labd struct {
	daemon  *daemon.Daemon
	seeder  *peer.Peer
	builder p2plab.Builder
	closers []io.Closer
}

func New(root, addr string, logger *zerolog.Logger, opts ...LabdOption) (*Labd, error) {
	var settings LabdSettings
	for _, opt := range opts {
		err := opt(&settings)
		if err != nil {
			return nil, err
		}
	}

	var closers []io.Closer
	db, err := metadata.NewDB(root)
	if err != nil {
		return nil, err
	}
	closers = append(closers, db)

	client, err := httputil.NewClient(httputil.NewHTTPClient(), httputil.WithLogger(logger))
	if err != nil {
		return nil, err
	}

	settings.ProviderSettings.DB = db
	settings.ProviderSettings.Logger = logger
	provider, err := providers.GetNodeProvider(filepath.Join(root, "providers"), settings.Provider, settings.ProviderSettings)
	if err != nil {
		return nil, err
	}

	settings.UploaderSettings.Client = client
	settings.UploaderSettings.Logger = logger
	uploader, err := uploaders.GetUploader(filepath.Join(root, "uploaders"), settings.Uploader, settings.UploaderSettings)
	if err != nil {
		return nil, err
	}
	closers = append(closers, uploader)

	builder, err := builder.New(filepath.Join(root, "builder"), db, uploader)
	if err != nil {
		return nil, err
	}

	sctx, cancel := context.WithCancel(context.Background())
	seeder, err := peer.New(sctx, filepath.Join(root, "seeder"), settings.Libp2pAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create seeder peer")
	}
	closers = append(closers, &daemon.CancelCloser{cancel})

	ts := transformers.New(filepath.Join(root, "transformers"), client.HTTPClient)
	closers = append(closers, ts)

	daemon, err := daemon.New("labd", addr, logger,
		healthcheckrouter.New(),
		clusterrouter.New(db, provider, client),
		noderouter.New(db, client),
		scenariorouter.New(db),
		benchmarkrouter.New(db, client, ts, seeder, builder),
		experimentrouter.New(db, provider, client, ts, seeder, builder),
	)
	if err != nil {
		return nil, err
	}
	closers = append(closers, daemon)

	d := &Labd{
		daemon:  daemon,
		seeder:  seeder,
		builder: builder,
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
	err := d.builder.Init(ctx)
	if err != nil {
		return err
	}
	zerolog.Ctx(ctx).Debug().Msg("Build initialized")

	var addrs []string
	for _, ma := range d.seeder.Host().Addrs() {
		addrs = append(addrs, ma.String())
	}
	zerolog.Ctx(ctx).Info().Strs("addrs", addrs).Msg("IPFS listening")

	return d.daemon.Serve(ctx)
}
