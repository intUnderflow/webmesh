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

// Package bridge contains a wrapper interface for running multiple mesh connections
// in parallel and sharing routes between them.
package bridge

import "github.com/webmeshproj/webmesh/pkg/mesh"

// Options are options for the bridge.
type Options struct {
	// Meshes are the meshes to bridge.
	Meshes map[string]*mesh.Options `json:",inline" yaml:",inline" toml:",inline"`
}