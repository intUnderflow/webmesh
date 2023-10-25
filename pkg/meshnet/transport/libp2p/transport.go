//go:build !wasm

/*
Copyright 2023 Avi Zimmerman <avi.zimmerman@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package libp2p

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	mnet "github.com/multiformats/go-multiaddr/net"
	"google.golang.org/grpc"

	"github.com/webmeshproj/webmesh/pkg/context"
	"github.com/webmeshproj/webmesh/pkg/meshnet/transport"
)

// TransportOptions are options for configuring an RPC transport over libp2p.
type TransportOptions struct {
	// Rendezvous is the pre-shared string to use as a rendezvous point for the DHT.
	Rendezvous string
	// HostOptions are options for configuring the host. These can be left
	// empty if using a pre-created host.
	HostOptions HostOptions
	// Host is a pre-started host to use for the transport.
	Host host.Host
}

// NewDiscoveryTransport returns a new RPC transport over libp2p using the IPFS DHT for
// discovery.
func NewDiscoveryTransport(ctx context.Context, opts TransportOptions) (transport.RPCTransport, error) {
	var h DiscoveryHost
	var err error
	var close func()
	if opts.Host != nil {
		dht, err := NewDHT(ctx, opts.Host, opts.HostOptions.BootstrapPeers, opts.HostOptions.ConnectTimeout)
		if err != nil {
			return nil, err
		}
		h = &discoveryHost{
			host: opts.Host,
			dht:  dht,
			opts: opts.HostOptions,
		}
		close = func() {
			err := dht.Close()
			if err != nil {
				context.LoggerFrom(ctx).Error("Failed to close DHT", "error", err.Error())
			}
		}
	} else {
		h, err = NewDiscoveryHost(ctx, opts.HostOptions)
		if err != nil {
			return nil, err
		}
		close = func() {
			err := h.Close(ctx)
			if err != nil {
				context.LoggerFrom(ctx).Error("Failed to close host", "error", err.Error())
			}
		}
	}
	return &rpcTransport{TransportOptions: opts, host: h, close: close}, nil
}

type rpcTransport struct {
	TransportOptions
	host  DiscoveryHost
	close func()
}

func (r *rpcTransport) Dial(ctx context.Context, address string) (*grpc.ClientConn, error) {
	log := context.LoggerFrom(ctx).With(slog.String("host-id", r.host.ID().String()))
	ctx = context.WithLogger(ctx, log)
	log.Debug("Searching for peers on the DHT with our PSK", slog.String("psk", r.Rendezvous))
	routingDiscovery := drouting.NewRoutingDiscovery(r.host.DHT())
	peerChan, err := routingDiscovery.FindPeers(ctx, r.Rendezvous)
	if err != nil {
		return nil, fmt.Errorf("libp2p find peers: %w", err)
	}
	// Wait for a peer to connect to
	log.Debug("Waiting for peer to establish connection with")
SearchPeers:
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("no peers found to dial: %w", ctx.Err())
		case peer, ok := <-peerChan:
			if !ok {
				if ctx.Err() != nil {
					return nil, fmt.Errorf("no peers found to dial: %w", ctx.Err())
				}
				continue SearchPeers
			}
			// Ignore ourselves and hosts with no addresses.
			jlog := log.With(slog.String("peer-id", peer.ID.String()), slog.Any("peer-addrs", peer.Addrs))
			if peer.ID == r.host.ID() || len(peer.Addrs) == 0 {
				jlog.Debug("Ignoring peer")
				continue
			}
			jlog.Debug("Dialing peer")
			var connCtx context.Context
			var cancel context.CancelFunc
			if r.HostOptions.ConnectTimeout > 0 {
				connCtx, cancel = context.WithTimeout(ctx, r.HostOptions.ConnectTimeout)
			} else {
				connCtx, cancel = context.WithCancel(ctx)
			}
			stream, err := r.host.Host().NewStream(connCtx, peer.ID, RPCProtocol)
			cancel()
			if err == nil {
				return grpc.DialContext(ctx, "", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
					return &streamConn{stream}, nil
				}))
			}
			jlog.Debug("Failed to dial peer", "error", err)

		}
	}
}

func (r *rpcTransport) Close() error {
	r.close()
	return nil
}

type streamConn struct {
	network.Stream
}

func (s *streamConn) LocalAddr() net.Addr {
	addr, _ := mnet.ToNetAddr(s.Stream.Conn().LocalMultiaddr())
	return addr
}

func (s *streamConn) RemoteAddr() net.Addr {
	addr, _ := mnet.ToNetAddr(s.Stream.Conn().RemoteMultiaddr())
	return addr
}