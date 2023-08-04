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

// Package meshbridge contains a wrapper interface for running multiple mesh connections
// in parallel and sharing routes between them.
package meshbridge

import (
	"fmt"

	"golang.org/x/exp/slog"

	"github.com/webmeshproj/webmesh/pkg/mesh"
)

// Bridge is the interface for a mesh bridge. It manages multiple mesh connections
// and services, sharing routes between them.
type Bridge interface{}

// New creates a new bridge.
func New(opts *Options) (Bridge, error) {
	err := opts.Validate()
	if err != nil {
		return nil, err
	}
	meshes := make(map[string]mesh.Mesh)
	for meshID, meshOpts := range opts.Meshes {
		id := meshID
		m, err := mesh.NewWithLogger(meshOpts.Mesh, slog.Default().With("mesh-id", id))
		if err != nil {
			return nil, fmt.Errorf("failed to create mesh %q: %w", id, err)
		}
		meshes[id] = m
	}
	return &meshBridge{opts: opts, meshes: meshes}, nil
}

type meshBridge struct {
	opts   *Options
	meshes map[string]mesh.Mesh
}