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

package storage

import (
	"bytes"
)

// Prefix is a prefix in the storage.
type Prefix []byte

var (
	// RegistryPrefix is the prefix for all data stored in the mesh registry.
	RegistryPrefix Prefix = []byte("/registry")

	// ConsensusPrefix is the prefix for all data stored related to consensus.
	ConsensusPrefix Prefix = []byte("/raft")
)

// String returns the string representation of the prefix.
func (p Prefix) String() string {
	return string(p)
}

// Contains returns true if the given key is contained in the prefix.
func (p Prefix) Contains(key []byte) bool {
	return bytes.HasPrefix(key, p)
}

// For is a helper method for creating a key for the prefix.
func (p Prefix) For(key []byte) Prefix {
	return Prefix(bytes.Join([][]byte{p, bytes.TrimPrefix(key, []byte("/"))}, []byte("/")))
}

// ForString is a helper method for creating a key for the prefix.
func (p Prefix) ForString(key string) Prefix {
	return p.For([]byte(key))
}

// TrimFrom returns the key without the prefix.
func (p Prefix) TrimFrom(key []byte) []byte {
	return bytes.TrimPrefix(key, append(p, '/'))
}

// ReservedPrefixes is a list of all reserved prefixes.
var ReservedPrefixes = []Prefix{
	RegistryPrefix,
	ConsensusPrefix,
}

// IsReservedPrefix returns true if the given key is reserved.
func IsReservedPrefix(key []byte) bool {
	for _, prefix := range ReservedPrefixes {
		if prefix.Contains(key) {
			return true
		}
	}
	return false
}
