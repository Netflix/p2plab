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
	"github.com/libp2p/go-libp2p-core/routing"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	mplex "github.com/libp2p/go-libp2p-mplex"
	secio "github.com/libp2p/go-libp2p-secio"
	tcp "github.com/libp2p/go-tcp-transport"
	ws "github.com/libp2p/go-ws-transport"
	multihash "github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func init() {
	ipld.Register(cid.DagProtobuf, dag.DecodeProtobufBlock)
	ipld.Register(cid.Raw, dag.DecodeRawBlock)
	ipld.Register(cid.DagCBOR, cbor.DecodeBlock) // need to decode CBOR
}

var (
	ReprovideInterval = 12 * time.Hour
)

type peer struct {
	h      host.Host
	dserv  ipld.DAGService
	system provider.System
	r      routing.ContentRouting
	bserv  blockservice.BlockService
	bs     blockstore.Blockstore
	ds     datastore.Batching
}

func NewPeer(ctx context.Context, root string) (p2plab.Peer, error) {
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
	return &peer{
		h:      h,
		dserv:  dserv,
		system: system,
		r:      r,
		bserv:  bserv,
		bs:     bs,
		ds:     ds,
	}, nil
}

func (p *peer) Host() host.Host {
	return p.h
}

func (p *peer) DAGService() ipld.DAGService {
	return p.dserv
}

func (p *peer) Provider() provider.System {
	return p.system
}

func (p *peer) ContentRouting() routing.ContentRouting {
	return p.r
}

func (p *peer) BlockService() blockservice.BlockService {
	return p.bserv
}

func (p *peer) Blockstore() blockstore.Blockstore {
	return p.bs
}

func (p *peer) Datastore() datastore.Batching {
	return p.ds
}

// Add chunks and adds content to the DAGService from a reader. The content
// is stored as a UnixFS DAG (default for IPFS). It returns the root
// ipld.Node.
func (p *peer) Add(ctx context.Context, r io.Reader, opts ...p2plab.AddOption) (ipld.Node, error) {
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
		Dagserv:    p.dserv,
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
func (p *peer) Get(ctx context.Context, c cid.Cid) (ufsio.ReadSeekCloser, error) {
	n, err := p.dserv.Get(ctx, c)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get file %q", c)
	}
	return ufsio.NewDagReader(ctx, n, p.dserv)
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
		"/ip4/0.0.0.0/tcp/0",
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
