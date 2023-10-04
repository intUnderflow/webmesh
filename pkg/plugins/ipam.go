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

// Package plugins contains the plugin manager.
package plugins

import (
	"fmt"
	"net/netip"
	"sync"

	v1 "github.com/webmeshproj/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/webmeshproj/webmesh/pkg/context"
	"github.com/webmeshproj/webmesh/pkg/storage"
)

// BuiltinIPAM is the built-in IPAM plugin that uses the mesh database
// to perform allocations.
type BuiltinIPAM struct {
	v1.UnimplementedIPAMPluginServer

	config  IPAMConfig
	db      storage.MeshDB
	datamux sync.Mutex
}

// IPAMConfig contains static address assignments for nodes.
type IPAMConfig struct {
	// StaticIPv4 is a map of node names to IPv4 addresses.
	StaticIPv4 map[string]any `mapstructure:"static-ipv4,omitempty" koanf:"static-ipv4,omitempty"`
}

// NewBuiltinIPAM returns a new ipam plugin with the given database.
func NewBuiltinIPAM(db storage.MeshDB) *BuiltinIPAM {
	return &BuiltinIPAM{db: db}
}

func (p *BuiltinIPAM) Allocate(ctx context.Context, r *v1.AllocateIPRequest, opts ...grpc.CallOption) (*v1.AllocatedIP, error) {
	p.datamux.Lock()
	defer p.datamux.Unlock()
	if addr, ok := p.config.StaticIPv4[r.GetNodeID()]; ok {
		return &v1.AllocatedIP{
			Ip: addr.(string),
		}, nil
	}
	return p.allocateV4(ctx, r)
}

func (p *BuiltinIPAM) Close(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (p *BuiltinIPAM) allocateV4(ctx context.Context, r *v1.AllocateIPRequest) (*v1.AllocatedIP, error) {
	globalPrefix, err := netip.ParsePrefix(r.GetSubnet())
	if err != nil {
		return nil, fmt.Errorf("parse subnet: %w", err)
	}
	nodes, err := p.db.Peers().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	allocated := make(map[netip.Prefix]struct{}, len(nodes))
	for _, node := range nodes {
		n := node
		if n.PrivateAddrV4().IsValid() {
			allocated[n.PrivateAddrV4()] = struct{}{}
		}
	}
	prefix, err := p.next32(globalPrefix, allocated)
	if err != nil {
		return nil, fmt.Errorf("find next available IPv4: %w", err)
	}
	return &v1.AllocatedIP{
		Ip: prefix.String(),
	}, nil
}

func (p *BuiltinIPAM) next32(cidr netip.Prefix, set map[netip.Prefix]struct{}) (netip.Prefix, error) {
	ip := cidr.Addr().Next()
	for cidr.Contains(ip) {
		prefix := netip.PrefixFrom(ip, 32)
		if _, ok := set[prefix]; !ok && !p.isStaticAllocation(prefix) {
			return prefix, nil
		}
		ip = ip.Next()
	}
	return netip.Prefix{}, fmt.Errorf("no more addresses in %s", cidr)
}

func (p *BuiltinIPAM) isStaticAllocation(ip netip.Prefix) bool {
	if ip.Addr().Is4() {
		for _, addr := range p.config.StaticIPv4 {
			if addr == ip.String() {
				return true
			}
		}
		return false
	}
	return false
}
