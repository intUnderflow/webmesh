/*
Copyright 2023.

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

// Package svcutil contains common utilities for services.
package svcutil

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/webmeshproj/api/v1"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	"github.com/webmeshproj/node/pkg/meshdb"
	"github.com/webmeshproj/node/pkg/meshdb/networking"
	"github.com/webmeshproj/node/pkg/meshdb/peers"
)

// PeerFromContext returns the peer ID from the context.
func PeerFromContext(ctx context.Context) (string, bool) {
	p, ok := peer.FromContext(ctx)
	if ok {
		if authInfo, ok := p.AuthInfo.(credentials.TLSInfo); ok {
			peerCerts := authInfo.State.PeerCertificates
			if len(peerCerts) > 0 {
				return peerCerts[0].Subject.CommonName, true
			}
		}
	}
	return "", false
}

// WireGuardPeersFor returns the WireGuard peers for the given peer ID.
// Peers are filtered by network ACLs.
func WireGuardPeersFor(ctx context.Context, store meshdb.Store, peerID string) ([]*v1.WireGuardPeer, error) {
	graph := peers.New(store).Graph()
	adjacencyMap, err := networking.New(store).FilterGraph(ctx, graph, peerID)
	if err != nil {
		return nil, fmt.Errorf("filter adjacency map: %w", err)
	}
	directAdjacents := adjacencyMap[peerID]
	out := make([]*v1.WireGuardPeer, 0, len(directAdjacents))
	for adjacent := range directAdjacents {
		node, err := graph.Vertex(adjacent)
		if err != nil {
			return nil, fmt.Errorf("get vertex: %w", err)
		}
		// Determine the preferred wireguard endpoint
		var primaryEndpoint string
		if node.PrimaryEndpoint != "" {
			for _, endpoint := range node.WireGuardEndpoints {
				if strings.HasPrefix(endpoint, node.PrimaryEndpoint) {
					primaryEndpoint = endpoint
					break
				}
			}
		}
		if primaryEndpoint == "" && len(node.WireGuardEndpoints) > 0 {
			primaryEndpoint = node.WireGuardEndpoints[0]
		}
		// Each direct adjacent is a peer
		peer := &v1.WireGuardPeer{
			Id:                 node.ID,
			PublicKey:          node.PublicKey.String(),
			ZoneAwarenessId:    node.ZoneAwarenessID,
			PrimaryEndpoint:    primaryEndpoint,
			WireguardEndpoints: node.WireGuardEndpoints,
			AddressIpv4: func() string {
				if node.PrivateIPv4.IsValid() {
					return node.PrivateIPv4.String()
				}
				return ""
			}(),
			AddressIpv6: func() string {
				if node.NetworkIPv6.IsValid() {
					return node.NetworkIPv6.String()
				}
				return ""
			}(),
		}
		allowedIPs, err := recurseAllowedIPs(graph, adjacencyMap, peerID, &node)
		if err != nil {
			return nil, fmt.Errorf("recurse allowed IPs: %w", err)
		}
		peer.AllowedIps = allowedIPs
		out = append(out, peer)
	}
	return out, nil
}

func recurseAllowedIPs(graph peers.Graph, adjacencyMap networking.AdjacencyMap, thisPeer string, node *peers.Node) ([]string, error) {
	allowedIPs := make([]string, 0)
	if node.PrivateIPv4.IsValid() {
		allowedIPs = append(allowedIPs, node.PrivateIPv4.String())
	}
	if node.NetworkIPv6.IsValid() {
		allowedIPs = append(allowedIPs, node.NetworkIPv6.String())
	}
	edgeIPs, err := recurseEdgeAllowedIPs(graph, adjacencyMap, thisPeer, node, nil)
	if err != nil {
		return nil, fmt.Errorf("recurse edge allowed IPs: %w", err)
	}
	for _, ip := range edgeIPs {
		if !contains(allowedIPs, ip) {
			allowedIPs = append(allowedIPs, ip)
		}
	}
	return allowedIPs, nil
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func recurseEdgeAllowedIPs(graph peers.Graph, adjacencyMap networking.AdjacencyMap, thisPeer string, node *peers.Node, visited map[string]struct{}) ([]string, error) {
	if visited == nil {
		visited = make(map[string]struct{})
	}
	allowedIPs := make([]string, 0)
	directAdjacents := adjacencyMap[thisPeer]
	visited[node.ID] = struct{}{}
	targets := adjacencyMap[node.ID]
	for target := range targets {
		if target == thisPeer {
			continue
		}
		if _, ok := directAdjacents[target]; ok {
			continue
		}
		if _, ok := visited[target]; !ok {
			targetNode, err := graph.Vertex(target)
			if err != nil {
				return nil, fmt.Errorf("get vertex: %w", err)
			}
			if targetNode.PrivateIPv4.IsValid() {
				allowedIPs = append(allowedIPs, targetNode.PrivateIPv4.String())
			}
			if targetNode.NetworkIPv6.IsValid() {
				allowedIPs = append(allowedIPs, targetNode.NetworkIPv6.String())
			}
			visited[target] = struct{}{}
			ips, err := recurseEdgeAllowedIPs(graph, adjacencyMap, thisPeer, &targetNode, visited)
			if err != nil {
				return nil, fmt.Errorf("recurse allowed IPs: %w", err)
			}
			allowedIPs = append(allowedIPs, ips...)
		}
	}
	return allowedIPs, nil
}
