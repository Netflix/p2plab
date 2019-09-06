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
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/dag"
	"github.com/Netflix/p2plab/metadata"
	bitswap "github.com/ipfs/go-bitswap"
	"github.com/ipfs/go-bitswap/network"
	blockservice "github.com/ipfs/go-blockservice"
	cid "github.com/ipfs/go-cid"
	datastore "github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	chunker "github.com/ipfs/go-ipfs-chunker"
	files "github.com/ipfs/go-ipfs-files"
	provider "github.com/ipfs/go-ipfs-provider"
	"github.com/ipfs/go-ipfs-provider/queue"
	"github.com/ipfs/go-ipfs-provider/simple"
	cbor "github.com/ipfs/go-ipld-cbor"
	ipld "github.com/ipfs/go-ipld-format"
	merkledag "github.com/ipfs/go-merkledag"
	unixfile "github.com/ipfs/go-unixfs/file"
	"github.com/ipfs/go-unixfs/importer/balanced"
	"github.com/ipfs/go-unixfs/importer/helpers"
	"github.com/ipfs/go-unixfs/importer/trickle"
	host "github.com/libp2p/go-libp2p-core/host"
	metrics "github.com/libp2p/go-libp2p-core/metrics"
	libp2ppeer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	swarm "github.com/libp2p/go-libp2p-swarm"
	filter "github.com/libp2p/go-maddr-filter"
	multiaddr "github.com/multiformats/go-multiaddr"
	multihash "github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

func init() {
	ipld.Register(cid.DagProtobuf, merkledag.DecodeProtobufBlock)
	ipld.Register(cid.Raw, merkledag.DecodeRawBlock)
	ipld.Register(cid.DagCBOR, cbor.DecodeBlock) // need to decode CBOR
}

var (
	ReprovideInterval = 12 * time.Hour
)

type Peer struct {
	host     host.Host
	dserv    ipld.DAGService
	system   provider.System
	r        routing.ContentRouting
	bswap    *bitswap.Bitswap
	bserv    blockservice.BlockService
	bs       blockstore.Blockstore
	ds       datastore.Batching
	swarm    *swarm.Swarm
	reporter metrics.Reporter
}

func New(ctx context.Context, root string, port int, pdef metadata.PeerDefinition) (*Peer, error) {
	ds, err := NewDatastore(root)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create datastore")
	}

	reporter := metrics.NewBandwidthCounter()
	h, r, err := NewLibp2pPeer(ctx, port, pdef, reporter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create libp2p peer")
	}

	swarm, ok := h.Network().(*swarm.Swarm)
	if !ok {
		return nil, errors.New("expected to be able to cast host network to swarm")
	}

	bs, err := NewBlockstore(ctx, ds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create blockstore")
	}

	bswapnet := network.NewFromIpfsHost(h, r)
	rem := bitswap.New(ctx, bswapnet, bs)

	bswap, ok := rem.(*bitswap.Bitswap)
	if !ok {
		return nil, errors.New("expected to be able to cast exchange interface to bitswap")
	}

	bserv := blockservice.New(bs, rem)

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

	dserv := merkledag.NewDAGService(bserv)
	return &Peer{
		host:     h,
		dserv:    dserv,
		system:   system,
		r:        r,
		bswap:    bswap,
		bserv:    bserv,
		bs:       bs,
		ds:       ds,
		swarm:    swarm,
		reporter: reporter,
	}, nil
}

func (p *Peer) Host() host.Host {
	return p.host
}

func (p *Peer) DAGService() ipld.DAGService {
	return p.dserv
}

func (p *Peer) Connect(ctx context.Context, infos []libp2ppeer.AddrInfo) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, info := range infos {
		info := info
		g.Go(func() error {
			ipnet, err := cidrFromAddrInfo(info)
			if err != nil {
				return err
			}
			p.swarm.Filters.Remove(ipnet)

			err = p.host.Connect(ctx, info)
			if err != nil && errors.Cause(err) != swarm.ErrDialToSelf {
				return err
			}
			return nil
		})
	}

	return g.Wait()
}

func (p *Peer) Disconnect(ctx context.Context, infos []libp2ppeer.AddrInfo) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, info := range infos {
		info := info
		g.Go(func() error {
			ipnet, err := cidrFromAddrInfo(info)
			if err != nil {
				return err
			}

			// Until libp2p has a disconnect protocol, we add a swarm filter for the
			// disconnected peer's CIDR to prevent reconnects after the connection is
			// dropped.
			p.swarm.Filters.AddFilter(*ipnet, filter.ActionDeny)

			err = p.host.Network().ClosePeer(info.ID)
			if err != nil {
				return err
			}

			return nil
		})
	}

	return g.Wait()
}

func (p *Peer) Add(ctx context.Context, r io.Reader, opts ...p2plab.AddOption) (ipld.Node, error) {
	settings := p2plab.AddSettings{
		Layout:    "balanced",
		Chunker:   "size-262144",
		RawLeaves: false,
		Hidden:    false,
		NoCopy:    false,
		HashFunc:  "sha2-256",
		MaxLinks:  helpers.DefaultLinksPerBlock,
	}
	for _, opt := range opts {
		err := opt(&settings)
		if err != nil {
			return nil, err
		}
	}

	prefix, err := merkledag.PrefixForCidVersion(1)
	if err != nil {
		return nil, errors.Wrap(err, "unrecognized CID version")
	}

	hashFuncCode, ok := multihash.Names[strings.ToLower(settings.HashFunc)]
	if !ok {
		return nil, errors.Wrapf(err, "unrecognized hash function %q", settings.HashFunc)
	}
	prefix.MhType = hashFuncCode

	dbp := helpers.DagBuilderParams{
		Dagserv:    p.dserv,
		RawLeaves:  settings.RawLeaves,
		Maxlinks:   settings.MaxLinks,
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

	var nd ipld.Node
	switch settings.Layout {
	case "trickle":
		nd, err = trickle.Layout(dbh)
	case "balanced":
		nd, err = balanced.Layout(dbh)
	default:
		return nil, errors.Errorf("unrecognized layout %q", settings.Layout)
	}

	return nd, err
}

func (p *Peer) FetchGraph(ctx context.Context, c cid.Cid) error {
	ng := merkledag.NewSession(ctx, p.dserv)
	return dag.Walk(ctx, c, ng)
}

func (p *Peer) Get(ctx context.Context, c cid.Cid) (files.Node, error) {
	nd, err := p.dserv.Get(ctx, c)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get file %q", c)
	}

	return unixfile.NewUnixfsFile(ctx, p.dserv, nd)
}

func (p *Peer) Report(ctx context.Context) (metadata.ReportNode, error) {
	stat, err := p.bswap.Stat()
	if err != nil {
		return metadata.ReportNode{}, err
	}

	return metadata.ReportNode{
		metadata.ReportBitswap{
			BlocksReceived:   stat.BlocksReceived,
			DataReceived:     stat.DataReceived,
			BlocksSent:       stat.BlocksSent,
			DataSent:         stat.DataSent,
			DupBlksReceived:  stat.DupBlksReceived,
			DupDataReceived:  stat.DupDataReceived,
			MessagesReceived: stat.MessagesReceived,
		},
		metadata.ReportBandwidth{
			Totals:    p.reporter.GetBandwidthTotals(),
			Peers:     p.reporter.GetBandwidthByPeer(),
			Protocols: p.reporter.GetBandwidthByProtocol(),
		},
	}, nil
}

func NewDatastore(path string) (datastore.Batching, error) {
	return badger.NewDatastore(path, &badger.DefaultOptions)
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

func cidrFromAddrInfo(info libp2ppeer.AddrInfo) (*net.IPNet, error) {
	if len(info.Addrs) == 0 {
		return nil, errors.New("addr info has zero addrs")
	}

	ma := info.Addrs[0]
	ip4Addr, err := ma.ValueForProtocol(multiaddr.ProtocolWithName("ip4").Code)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ip4 value")
	}

	_, ipnet, err := net.ParseCIDR(fmt.Sprintf("%s/32", ip4Addr))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse peer cidr")
	}

	return ipnet, nil
}
