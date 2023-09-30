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

package state

import (
	"context"
	"testing"

	v1 "github.com/webmeshproj/api/v1"

	"github.com/webmeshproj/webmesh/pkg/crypto"
	"github.com/webmeshproj/webmesh/pkg/storage/meshdb/peers"
	"github.com/webmeshproj/webmesh/pkg/storage/providers/backends/badgerdb"
)

var (
	ipv6Prefix = "fd00:dead::/48"
	ipv4Prefix = "172.16.0.0/12"
	domain     = "webmesh.internal"

	publicNode  = "public"
	privateNode = "private"

	publicNodePublicAddr = "1.1.1.1"

	publicNodePrivateAddr  = "172.16.0.1/32"
	privateNodePrivateAddr = "172.16.0.2/32"

	rpcPort = 1
)

func TestGetIPv6Prefix(t *testing.T) {
	t.Parallel()

	state, teardown := setupTest(t)
	defer teardown()
	prefix, err := state.GetIPv6Prefix(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if prefix.String() != ipv6Prefix {
		t.Fatalf("expected %s, got %s", ipv6Prefix, prefix)
	}
}

func TestGetIPv4Prefix(t *testing.T) {
	t.Parallel()

	state, teardown := setupTest(t)
	defer teardown()
	prefix, err := state.GetIPv4Prefix(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if prefix.String() != ipv4Prefix {
		t.Fatalf("expected %s, got %s", ipv4Prefix, prefix)
	}
}

func TestGetMeshDomain(t *testing.T) {
	t.Parallel()

	state, teardown := setupTest(t)
	defer teardown()
	got, err := state.GetMeshDomain(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if domain != got {
		t.Fatalf("expected %q, got %s", domain, got)
	}
}

func setupTest(t *testing.T) (*state, func()) {
	t.Helper()
	db, err := badgerdb.NewInMemory(badgerdb.Options{})
	if err != nil {
		t.Fatalf("create test db: %v", err)
	}
	close := func() {
		err := db.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	err = db.PutValue(ctx, IPv6PrefixKey, []byte(ipv6Prefix), 0)
	if err != nil {
		t.Fatal(err)
	}
	err = db.PutValue(ctx, IPv4PrefixKey, []byte(ipv4Prefix), 0)
	if err != nil {
		t.Fatal(err)
	}
	err = db.PutValue(ctx, MeshDomainKey, []byte(domain), 0)
	if err != nil {
		t.Fatal(err)
	}
	p := peers.New(db)
	// Node with public address
	err = p.Put(ctx, &v1.MeshNode{
		Id:              publicNode,
		PublicKey:       mustGenerateKey(t),
		PrimaryEndpoint: publicNodePublicAddr,
		PrivateIpv4:     publicNodePrivateAddr,
		Features: []*v1.FeaturePort{
			{
				Feature: v1.Feature_NODES,
				Port:    int32(rpcPort),
			},
			{
				Feature: v1.Feature_STORAGE_PROVIDER,
				Port:    2,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Node with private address
	err = p.Put(ctx, &v1.MeshNode{
		Id:          privateNode,
		PublicKey:   mustGenerateKey(t),
		PrivateIpv4: privateNodePrivateAddr,
		Features: []*v1.FeaturePort{
			{
				Feature: v1.Feature_NODES,
				Port:    int32(rpcPort),
			},
			{
				Feature: v1.Feature_STORAGE_PROVIDER,
				Port:    2,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := New(db)
	return s.(*state), close
}

func mustGenerateKey(t *testing.T) string {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := key.PublicKey().Encode()
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}