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

	noise "github.com/ChainSafe/go-libp2p-noise"
	"github.com/Netflix/p2plab/errdefs"
	"github.com/Netflix/p2plab/metadata"
	nilrouting "github.com/ipfs/go-ipfs-routing/none"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	host "github.com/libp2p/go-libp2p-core/host"
	metrics "github.com/libp2p/go-libp2p-core/metrics"
	libp2ppeer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	mplex "github.com/libp2p/go-libp2p-mplex"
	quic "github.com/libp2p/go-libp2p-quic-transport"
	secio "github.com/libp2p/go-libp2p-secio"
	tls "github.com/libp2p/go-libp2p-tls"
	yamux "github.com/libp2p/go-libp2p-yamux"
	tcp "github.com/libp2p/go-tcp-transport"
	ws "github.com/libp2p/go-ws-transport"
	"github.com/pkg/errors"
)

func NewLibp2pPeer(ctx context.Context, port int, pdef metadata.PeerDefinition, reporter metrics.Reporter) (host.Host, routing.ContentRouting, error) {
	var (
		addresses        []string
		transportOptions []libp2p.Option
	)
	for _, transportType := range pdef.Transports {
		option, address, err := NewTransportOption(transportType, port)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create transport option")
		}
		addresses = append(addresses, address)
		transportOptions = append(transportOptions, option)
	}

	var muxerOptions []libp2p.Option
	for _, muxerType := range pdef.Muxers {
		option, err := NewMuxerOption(muxerType)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create muxer option")
		}
		muxerOptions = append(muxerOptions, option)
	}

	var securityOptions []libp2p.Option
	for _, securityType := range pdef.SecurityTransports {
		option, err := NewSecurityOption(securityType)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create security option")
		}
		securityOptions = append(securityOptions, option)
	}

	routingOption, r, err := NewRoutingOption(ctx, pdef.Routing)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create routing option")
	}

	host, err := libp2p.New(
		ctx,
		libp2p.ListenAddrStrings(addresses...),
		libp2p.ChainOptions(transportOptions...),
		libp2p.ChainOptions(muxerOptions...),
		libp2p.ChainOptions(securityOptions...),
		libp2p.BandwidthReporter(reporter),
		routingOption,
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create libp2p host")
	}

	return host, r, nil
}

func NewTransportOption(transportType string, port int) (libp2p.Option, string, error) {
	switch transportType {
	case "tcp":
		return libp2p.Transport(tcp.NewTCPTransport), fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port), nil
	case "ws":
		return libp2p.Transport(ws.New), fmt.Sprintf("/ip4/0.0.0.0/tcp/%d/ws", port), nil
	case "quic":
		return libp2p.Transport(quic.NewTransport), fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", port), nil
	default:
		return nil, "", errors.Wrapf(errdefs.ErrInvalidArgument, "transport %q", transportType)
	}
}

func NewMuxerOption(muxerType string) (libp2p.Option, error) {
	switch muxerType {
	case "mplex":
		return libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport), nil
	case "yamux":
		return libp2p.Muxer("/yamux/1.0.0", yamux.DefaultTransport), nil
	default:
		return nil, errors.Wrapf(errdefs.ErrInvalidArgument, "muxer %q", muxerType)
	}
}

func NewSecurityOption(securityType string) (libp2p.Option, error) {
	switch securityType {
	case "tls":
		return libp2p.Security(tls.ID, tls.New), nil
	case "secio":
		return libp2p.Security(secio.ID, secio.New), nil
	case "noise":
		return libp2p.Security(noise.ID, NewNoise), nil
	default:
		return nil, errors.Wrapf(errdefs.ErrInvalidArgument, "security %q", securityType)
	}
}

func NewNoise(sk crypto.PrivKey, pid libp2ppeer.ID) (*noise.Transport, error) {
	return noise.NewTransport(pid, sk, false, nil), nil
}

func NewRoutingOption(ctx context.Context, routingType string) (libp2p.Option, routing.ContentRouting, error) {
	switch routingType {
	case "nil":
		routing, err := nilrouting.ConstructNilRouting(nil, nil, nil, nil)
		if err != nil {
			return nil, nil, err
		}
		return libp2p.Routing(nil), routing, nil
	case "kaddht":
		var dht *kaddht.IpfsDHT
		newDHT := func(h host.Host) (routing.PeerRouting, error) {
			var err error
			dht, err = kaddht.New(ctx, h)
			return dht, err
		}
		return libp2p.Routing(newDHT), dht, nil
	default:
		return nil, nil, errors.Wrapf(errdefs.ErrInvalidArgument, "routing %q", routingType)
	}
}
