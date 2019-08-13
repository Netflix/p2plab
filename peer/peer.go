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

package peer

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/Netflix/p2plab"
	bitswap "github.com/ipfs/go-bitswap"
	"github.com/ipfs/go-bitswap/network"
	blockservice "github.com/ipfs/go-blockservice"
	cid "github.com/ipfs/go-cid"
	datastore "github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	chunker "github.com/ipfs/go-ipfs-chunker"
	provider "github.com/ipfs/go-ipfs-provider"
	"github.com/ipfs/go-ipfs-provider/queue"
	"github.com/ipfs/go-ipfs-provider/simple"
	cbor "github.com/ipfs/go-ipld-cbor"
	ipld "github.com/ipfs/go-ipld-format"
	dag "github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-unixfs/importer/balanced"
	"github.com/ipfs/go-unixfs/importer/helpers"
	"github.com/ipfs/go-unixfs/importer/trickle"
	ufsio "github.com/ipfs/go-unixfs/io"
	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-core/host"
	libp2ppeer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	mplex "github.com/libp2p/go-libp2p-mplex"
	secio "github.com/libp2p/go-libp2p-secio"
	tcp "github.com/libp2p/go-tcp-transport"
	ws "github.com/libp2p/go-ws-transport"
	multihash "github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

func init() {
	ipld.Register(cid.DagProtobuf, dag.DecodeProtobufBlock)
	ipld.Register(cid.Raw, dag.DecodeRawBlock)
	ipld.Register(cid.DagCBOR, cbor.DecodeBlock) // need to decode CBOR
}

var (
	ReprovideInterval = 12 * time.Hour
)

type Peer struct {
	host.Host
	ipld.DAGService
	provider.System
	routing.ContentRouting
	blockservice.BlockService
	blockstore.Blockstore
	datastore.Batching
}

func New(ctx context.Context, root string) (*Peer, error) {
	ds, err := NewDatastore(root)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create datastore")
	}

	h, r, err := NewLibp2pPeer(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create libp2p peer")
	}

	bs, err := NewBlockstore(ctx, ds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create blockstore")
	}

	bserv := NewBlockService(ctx, bs, h, r)

	system, err := NewProviderSystem(ctx, ds, bs, r)
	if err != nil {
		bserv.Close()
		return nil, errors.Wrap(err, "failed to create provider system")
	}

	go func() {
		system.Run()

		select {
		case <-ctx.Done():
			err := system.Close()
			if err != nil {
				log.Warn().Msgf("failed to close provider: %q", err)
			}

			err = bserv.Close()
			if err != nil {
				log.Warn().Msgf("failed to close block service: %q", err)
			}
		}
	}()

	dserv := dag.NewDAGService(bserv)
	return &Peer{
		Host:           h,
		DAGService:     dserv,
		System:         system,
		ContentRouting: r,
		BlockService:   bserv,
		Blockstore:     bs,
		Batching:       ds,
	}, nil
}

func (p *Peer) Connect(ctx context.Context, infos []libp2ppeer.AddrInfo) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, info := range infos {
		g.Go(func() error {
			return p.Host.Connect(ctx, info)
		})
	}

	return g.Wait()
}

func (p *Peer) Disconnect(ctx context.Context, ids []libp2ppeer.ID) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, id := range ids {
		g.Go(func() error {
			return p.Host.Network().ClosePeer(id)
		})
	}

	return g.Wait()
}

// Add chunks and adds content to the DAGService from a reader. The content
// is stored as a UnixFS DAG (default for IPFS). It returns the root
// ipld.Node.
func (p *Peer) Add(ctx context.Context, r io.Reader, opts ...p2plab.AddOption) (ipld.Node, error) {
	settings := p2plab.AddSettings{
		Layout:    "balanced",
		Chunker:   "size-262144",
		RawLeaves: false,
		Hidden:    false,
		NoCopy:    false,
		HashFunc:  "sha2-256",
	}
	for _, opt := range opts {
		err := opt(&settings)
		if err != nil {
			return nil, err
		}
	}

	prefix, err := dag.PrefixForCidVersion(1)
	if err != nil {
		return nil, errors.Wrap(err, "unrecognized CID version")
	}

	hashFuncCode, ok := multihash.Names[strings.ToLower(settings.HashFunc)]
	if !ok {
		return nil, errors.Wrapf(err, "unrecognized hash function %q", settings.HashFunc)
	}
	prefix.MhType = hashFuncCode

	dbp := helpers.DagBuilderParams{
		Dagserv:    p.DAGService,
		RawLeaves:  settings.RawLeaves,
		Maxlinks:   helpers.DefaultLinksPerBlock,
		NoCopy:     settings.NoCopy,
		CidBuilder: &prefix,
	}

	chnk, err := chunker.FromString(r, settings.Chunker)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create chunker")
	}

	dbh, err := dbp.New(chnk)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create dag builder")
	}

	var n ipld.Node
	switch settings.Layout {
	case "trickle":
		n, err = trickle.Layout(dbh)
	case "balanced":
		n, err = balanced.Layout(dbh)
	default:
		return nil, errors.Errorf("unrecognized layout %q", settings.Layout)
	}

	return n, err
}

// Get returns a reader to a file as identified by its root CID. The file
// must have been added as a UnixFS DAG (default for IPFS).
func (p *Peer) Get(ctx context.Context, c cid.Cid) (ufsio.ReadSeekCloser, error) {
	n, err := p.DAGService.Get(ctx, c)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get file %q", c)
	}
	return ufsio.NewDagReader(ctx, n, p.DAGService)
}

func NewDatastore(path string) (datastore.Batching, error) {
	return badger.NewDatastore(path, &badger.DefaultOptions)
}

func NewLibp2pPeer(ctx context.Context) (host.Host, routing.ContentRouting, error) {
	transports := libp2p.ChainOptions(
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(ws.New),
	)

	muxers := libp2p.ChainOptions(
		libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport),
	)

	security := libp2p.Security(secio.ID, secio.New)

	listenAddrs := libp2p.ListenAddrStrings(
		"/ip4/0.0.0.0/tcp/4001",
	)

	var dht *kaddht.IpfsDHT
	newDHT := func(h host.Host) (routing.PeerRouting, error) {
		var err error
		dht, err = kaddht.New(ctx, h)
		return dht, err
	}
	routing := libp2p.Routing(newDHT)

	host, err := libp2p.New(
		ctx,
		transports,
		listenAddrs,
		muxers,
		security,
		routing,
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create libp2p host")
	}

	return host, dht, nil
}

func NewBlockstore(ctx context.Context, ds datastore.Batching) (blockstore.Blockstore, error) {
	bs := blockstore.NewBlockstore(ds)
	bs = blockstore.NewIdStore(bs)

	// var err error
	// bs, err = blockstore.CachedBlockstore(ctx, bs, blockstore.DefaultCacheOpts())
	// if err != nil {
	// 	return nil, err
	// }

	return bs, nil
}

func NewBlockService(ctx context.Context, bs blockstore.Blockstore, h host.Host, r routing.ContentRouting) blockservice.BlockService {
	bswapnet := network.NewFromIpfsHost(h, r)
	rem := bitswap.New(ctx, bswapnet, bs)
	return blockservice.New(bs, rem)
}

func NewProviderSystem(ctx context.Context, ds datastore.Batching, bs blockstore.Blockstore, r routing.ContentRouting) (provider.System, error) {
	queue, err := queue.NewQueue(ctx, "repro", ds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new queue")
	}

	prov := simple.NewProvider(ctx, queue, r)
	reprov := simple.NewReprovider(ctx, ReprovideInterval, r, simple.NewBlockstoreProvider(bs))
	system := provider.NewSystem(prov, reprov)
	return system, nil
}
