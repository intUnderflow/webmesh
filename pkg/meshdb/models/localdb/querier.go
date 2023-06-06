// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.18.0

package localdb

import (
	"context"
)

type Querier interface {
	DropRaftServers(ctx context.Context) error
	GetCurrentRaftIndex(ctx context.Context) (RaftIndex, error)
	GetCurrentWireguardKey(ctx context.Context) (WireguardKey, error)
	GetRaftServers(ctx context.Context) ([]RaftServer, error)
	InsertRaftServer(ctx context.Context, arg InsertRaftServerParams) error
	SetCurrentRaftIndex(ctx context.Context, arg SetCurrentRaftIndexParams) error
	SetCurrentWireguardKey(ctx context.Context, arg SetCurrentWireguardKeyParams) error
}

var _ Querier = (*Queries)(nil)
