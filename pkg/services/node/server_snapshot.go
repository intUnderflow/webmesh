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

package node

import (
	"context"
	"io"

	v1 "github.com/webmeshproj/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) Snapshot(ctx context.Context, req *v1.SnapshotRequest) (*v1.SnapshotResponse, error) {
	f := s.store.Raft().Snapshot()
	if err := f.Error(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create snapshot: %v", err)
	}
	_, r, err := f.Open()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to open snapshot: %v", err)
	}
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read snapshot: %v", err)
	}
	return &v1.SnapshotResponse{Snapshot: data}, nil
}
