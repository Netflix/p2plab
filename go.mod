module github.com/Netflix/p2plab

go 1.12

require (
	github.com/Microsoft/go-winio v0.4.13-0.20190408173621-84b4ab48a507 // indirect
	github.com/Microsoft/hcsshim v0.8.6 // indirect
	github.com/alecthomas/template v0.0.0-20160405071501-a0175ee3bccc
	github.com/aws/aws-sdk-go-v2 v0.11.0
	github.com/codahale/hdrhistogram v0.0.0-20160425231609-f8ad88b59a58 // indirect
	github.com/containerd/containerd v1.3.0-0.20190507210959-7c1e88399ec0
	github.com/containerd/continuity v0.0.0-20190426062206-aaeac12a7ffc // indirect
	github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/gobwas/glob v0.2.3
	github.com/gorilla/mux v1.7.3
	github.com/hako/durafmt v0.0.0-20190612201238-650ed9f29a84
	github.com/hashicorp/go-cleanhttp v0.5.0
	github.com/hashicorp/go-retryablehttp v0.5.4
	github.com/ipfs/go-bitswap v0.1.6
	github.com/ipfs/go-blockservice v0.1.0
	github.com/ipfs/go-cid v0.0.3
	github.com/ipfs/go-datastore v0.1.0
	github.com/ipfs/go-ds-badger v0.0.5
	github.com/ipfs/go-ipfs v0.4.22-0.20190826175208-27c0de7f7ceb
	github.com/ipfs/go-ipfs-blockstore v0.1.0
	github.com/ipfs/go-ipfs-chunker v0.0.1
	github.com/ipfs/go-ipfs-files v0.0.3
	github.com/ipfs/go-ipfs-provider v0.2.1
	github.com/ipfs/go-ipfs-routing v0.1.0
	github.com/ipfs/go-ipfs-util v0.0.1
	github.com/ipfs/go-ipld-cbor v0.0.2
	github.com/ipfs/go-ipld-format v0.0.2
	github.com/ipfs/go-merkledag v0.2.3
	github.com/ipfs/go-unixfs v0.2.1
	github.com/libp2p/go-libp2p v0.3.0
	github.com/libp2p/go-libp2p-core v0.2.2
	github.com/libp2p/go-libp2p-mplex v0.2.1
	github.com/libp2p/go-libp2p-peer v0.2.0
	github.com/libp2p/go-libp2p-peerstore v0.1.3
	github.com/libp2p/go-libp2p-protocol v0.1.0
	github.com/libp2p/go-libp2p-secio v0.2.0
	github.com/libp2p/go-libp2p-swarm v0.2.1
	github.com/libp2p/go-maddr-filter v0.0.5
	github.com/libp2p/go-tcp-transport v0.1.0
	github.com/libp2p/go-ws-transport v0.1.0
	github.com/multiformats/go-multiaddr v0.0.4
	github.com/multiformats/go-multihash v0.0.7
	github.com/olekukonko/tablewriter v0.0.2-0.20190618033246-cc27d85e17ce
	github.com/opencontainers/go-digest v1.0.0-rc1
	github.com/opencontainers/image-spec v1.0.1
	github.com/opencontainers/runc v1.0.0-rc8 // indirect
	github.com/opentracing-contrib/go-stdlib v0.0.0-20190519235532-cf7a6c988dc9
	github.com/opentracing/opentracing-go v1.1.0
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/errors v0.8.1
	github.com/rs/xid v1.2.1
	github.com/rs/zerolog v1.14.4-0.20190719171043-b806a5ecbe53
	github.com/sirupsen/logrus v1.4.0 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/uber-go/atomic v1.4.0 // indirect
	github.com/uber/jaeger-client-go v2.16.0+incompatible
	github.com/uber/jaeger-lib v2.0.0+incompatible // indirect
	github.com/urfave/cli v1.20.0
	go.etcd.io/bbolt v1.3.3
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	google.golang.org/grpc v1.20.1 // indirect
	gotest.tools v2.2.0+incompatible // indirect
)

replace github.com/ipfs/go-bitswap => github.com/hinshun/go-bitswap v0.1.7-0.20190827013313-3514b1ab1d4e
