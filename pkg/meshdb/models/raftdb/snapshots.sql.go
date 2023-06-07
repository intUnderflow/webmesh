// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.18.0
// source: snapshots.sql

package raftdb

import (
	"context"
	"database/sql"
	"time"
)

const DropGroups = `-- name: DropGroups :exec
DELETE FROM groups
`

func (q *Queries) DropGroups(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, DropGroups)
	return err
}

const DropLeases = `-- name: DropLeases :exec
DELETE FROM leases
`

func (q *Queries) DropLeases(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, DropLeases)
	return err
}

const DropMeshState = `-- name: DropMeshState :exec
DELETE FROM mesh_state
`

func (q *Queries) DropMeshState(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, DropMeshState)
	return err
}

const DropNetworkACLs = `-- name: DropNetworkACLs :exec
DELETE FROM network_acls
`

func (q *Queries) DropNetworkACLs(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, DropNetworkACLs)
	return err
}

const DropNodeEdges = `-- name: DropNodeEdges :exec
DELETE FROM node_edges
`

func (q *Queries) DropNodeEdges(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, DropNodeEdges)
	return err
}

const DropNodes = `-- name: DropNodes :exec
DELETE FROM nodes
`

func (q *Queries) DropNodes(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, DropNodes)
	return err
}

const DropRoleBindings = `-- name: DropRoleBindings :exec
DELETE FROM role_bindings
`

func (q *Queries) DropRoleBindings(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, DropRoleBindings)
	return err
}

const DropRoles = `-- name: DropRoles :exec
DELETE FROM roles
`

func (q *Queries) DropRoles(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, DropRoles)
	return err
}

const DropUsers = `-- name: DropUsers :exec
DELETE FROM users
`

func (q *Queries) DropUsers(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, DropUsers)
	return err
}

const DumpGroups = `-- name: DumpGroups :many
SELECT name, users, nodes, created_at, updated_at FROM groups
`

func (q *Queries) DumpGroups(ctx context.Context) ([]Group, error) {
	rows, err := q.db.QueryContext(ctx, DumpGroups)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Group
	for rows.Next() {
		var i Group
		if err := rows.Scan(
			&i.Name,
			&i.Users,
			&i.Nodes,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const DumpLeases = `-- name: DumpLeases :many
SELECT node_id, ipv4, created_at FROM leases
`

func (q *Queries) DumpLeases(ctx context.Context) ([]Lease, error) {
	rows, err := q.db.QueryContext(ctx, DumpLeases)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Lease
	for rows.Next() {
		var i Lease
		if err := rows.Scan(&i.NodeID, &i.Ipv4, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const DumpMeshState = `-- name: DumpMeshState :many
SELECT "key", value FROM mesh_state
`

func (q *Queries) DumpMeshState(ctx context.Context) ([]MeshState, error) {
	rows, err := q.db.QueryContext(ctx, DumpMeshState)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []MeshState
	for rows.Next() {
		var i MeshState
		if err := rows.Scan(&i.Key, &i.Value); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const DumpNetworkACLs = `-- name: DumpNetworkACLs :many
SELECT name, src_node_ids, dst_node_ids, src_cidrs, dst_cidrs, "action", created_at, updated_at FROM network_acls
`

func (q *Queries) DumpNetworkACLs(ctx context.Context) ([]NetworkAcl, error) {
	rows, err := q.db.QueryContext(ctx, DumpNetworkACLs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []NetworkAcl
	for rows.Next() {
		var i NetworkAcl
		if err := rows.Scan(
			&i.Name,
			&i.SrcNodeIds,
			&i.DstNodeIds,
			&i.SrcCidrs,
			&i.DstCidrs,
			&i.Action,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const DumpNodeEdges = `-- name: DumpNodeEdges :many
SELECT src_node_id, dst_node_id, weight, attrs FROM node_edges
`

func (q *Queries) DumpNodeEdges(ctx context.Context) ([]NodeEdge, error) {
	rows, err := q.db.QueryContext(ctx, DumpNodeEdges)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []NodeEdge
	for rows.Next() {
		var i NodeEdge
		if err := rows.Scan(
			&i.SrcNodeID,
			&i.DstNodeID,
			&i.Weight,
			&i.Attrs,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const DumpNodes = `-- name: DumpNodes :many
SELECT id, public_key, raft_port, grpc_port, primary_endpoint, wireguard_endpoints, zone_awareness_id, network_ipv6, created_at, updated_at FROM nodes
`

func (q *Queries) DumpNodes(ctx context.Context) ([]Node, error) {
	rows, err := q.db.QueryContext(ctx, DumpNodes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Node
	for rows.Next() {
		var i Node
		if err := rows.Scan(
			&i.ID,
			&i.PublicKey,
			&i.RaftPort,
			&i.GrpcPort,
			&i.PrimaryEndpoint,
			&i.WireguardEndpoints,
			&i.ZoneAwarenessID,
			&i.NetworkIpv6,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const DumpRoleBindings = `-- name: DumpRoleBindings :many
SELECT name, role_name, node_ids, user_names, group_names, created_at, updated_at FROM role_bindings
`

func (q *Queries) DumpRoleBindings(ctx context.Context) ([]RoleBinding, error) {
	rows, err := q.db.QueryContext(ctx, DumpRoleBindings)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []RoleBinding
	for rows.Next() {
		var i RoleBinding
		if err := rows.Scan(
			&i.Name,
			&i.RoleName,
			&i.NodeIds,
			&i.UserNames,
			&i.GroupNames,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const DumpRoles = `-- name: DumpRoles :many
SELECT name, rules_json, created_at, updated_at FROM roles
`

func (q *Queries) DumpRoles(ctx context.Context) ([]Role, error) {
	rows, err := q.db.QueryContext(ctx, DumpRoles)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Role
	for rows.Next() {
		var i Role
		if err := rows.Scan(
			&i.Name,
			&i.RulesJson,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const DumpUsers = `-- name: DumpUsers :many
SELECT name, created_at, updated_at FROM users
`

func (q *Queries) DumpUsers(ctx context.Context) ([]User, error) {
	rows, err := q.db.QueryContext(ctx, DumpUsers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []User
	for rows.Next() {
		var i User
		if err := rows.Scan(&i.Name, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const RestoreGroup = `-- name: RestoreGroup :exec
INSERT INTO groups (
    name,
    users,
    nodes,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?)
`

type RestoreGroupParams struct {
	Name      string         `json:"name"`
	Users     sql.NullString `json:"users"`
	Nodes     sql.NullString `json:"nodes"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

func (q *Queries) RestoreGroup(ctx context.Context, arg RestoreGroupParams) error {
	_, err := q.db.ExecContext(ctx, RestoreGroup,
		arg.Name,
		arg.Users,
		arg.Nodes,
		arg.CreatedAt,
		arg.UpdatedAt,
	)
	return err
}

const RestoreLease = `-- name: RestoreLease :exec
INSERT INTO leases (
    node_id,
    ipv4,
    created_at
) VALUES ( ?, ?, ? )
`

type RestoreLeaseParams struct {
	NodeID    string    `json:"node_id"`
	Ipv4      string    `json:"ipv4"`
	CreatedAt time.Time `json:"created_at"`
}

func (q *Queries) RestoreLease(ctx context.Context, arg RestoreLeaseParams) error {
	_, err := q.db.ExecContext(ctx, RestoreLease, arg.NodeID, arg.Ipv4, arg.CreatedAt)
	return err
}

const RestoreMeshState = `-- name: RestoreMeshState :exec
INSERT INTO mesh_state (key, value) VALUES (?, ?)
`

type RestoreMeshStateParams struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (q *Queries) RestoreMeshState(ctx context.Context, arg RestoreMeshStateParams) error {
	_, err := q.db.ExecContext(ctx, RestoreMeshState, arg.Key, arg.Value)
	return err
}

const RestoreNetworkACL = `-- name: RestoreNetworkACL :exec
INSERT INTO network_acls (
    name,
    src_node_ids,
    dst_node_ids,
    src_cidrs,
    dst_cidrs,
    action,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`

type RestoreNetworkACLParams struct {
	Name       string         `json:"name"`
	SrcNodeIds sql.NullString `json:"src_node_ids"`
	DstNodeIds sql.NullString `json:"dst_node_ids"`
	SrcCidrs   sql.NullString `json:"src_cidrs"`
	DstCidrs   sql.NullString `json:"dst_cidrs"`
	Action     int64          `json:"action"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

func (q *Queries) RestoreNetworkACL(ctx context.Context, arg RestoreNetworkACLParams) error {
	_, err := q.db.ExecContext(ctx, RestoreNetworkACL,
		arg.Name,
		arg.SrcNodeIds,
		arg.DstNodeIds,
		arg.SrcCidrs,
		arg.DstCidrs,
		arg.Action,
		arg.CreatedAt,
		arg.UpdatedAt,
	)
	return err
}

const RestoreNode = `-- name: RestoreNode :exec
INSERT INTO nodes (
    id,
    public_key,
    raft_port,
    grpc_port,
    primary_endpoint,
    wireguard_endpoints,
    zone_awareness_id,
    network_ipv6,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

type RestoreNodeParams struct {
	ID                 string         `json:"id"`
	PublicKey          sql.NullString `json:"public_key"`
	RaftPort           int64          `json:"raft_port"`
	GrpcPort           int64          `json:"grpc_port"`
	PrimaryEndpoint    sql.NullString `json:"primary_endpoint"`
	WireguardEndpoints sql.NullString `json:"wireguard_endpoints"`
	ZoneAwarenessID    sql.NullString `json:"zone_awareness_id"`
	NetworkIpv6        sql.NullString `json:"network_ipv6"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

func (q *Queries) RestoreNode(ctx context.Context, arg RestoreNodeParams) error {
	_, err := q.db.ExecContext(ctx, RestoreNode,
		arg.ID,
		arg.PublicKey,
		arg.RaftPort,
		arg.GrpcPort,
		arg.PrimaryEndpoint,
		arg.WireguardEndpoints,
		arg.ZoneAwarenessID,
		arg.NetworkIpv6,
		arg.CreatedAt,
		arg.UpdatedAt,
	)
	return err
}

const RestoreNodeEdge = `-- name: RestoreNodeEdge :exec
INSERT INTO node_edges (src_node_id, dst_node_id, weight, attrs) VALUES (?, ?, ?, ?)
`

type RestoreNodeEdgeParams struct {
	SrcNodeID string         `json:"src_node_id"`
	DstNodeID string         `json:"dst_node_id"`
	Weight    int64          `json:"weight"`
	Attrs     sql.NullString `json:"attrs"`
}

func (q *Queries) RestoreNodeEdge(ctx context.Context, arg RestoreNodeEdgeParams) error {
	_, err := q.db.ExecContext(ctx, RestoreNodeEdge,
		arg.SrcNodeID,
		arg.DstNodeID,
		arg.Weight,
		arg.Attrs,
	)
	return err
}

const RestoreRole = `-- name: RestoreRole :exec
INSERT INTO roles (
    name,
    rules_json,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?)
`

type RestoreRoleParams struct {
	Name      string    `json:"name"`
	RulesJson string    `json:"rules_json"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (q *Queries) RestoreRole(ctx context.Context, arg RestoreRoleParams) error {
	_, err := q.db.ExecContext(ctx, RestoreRole,
		arg.Name,
		arg.RulesJson,
		arg.CreatedAt,
		arg.UpdatedAt,
	)
	return err
}

const RestoreRoleBinding = `-- name: RestoreRoleBinding :exec
INSERT INTO role_bindings (
    name,
    role_name,
    node_ids,
    user_names,
    group_names,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
`

type RestoreRoleBindingParams struct {
	Name       string         `json:"name"`
	RoleName   string         `json:"role_name"`
	NodeIds    sql.NullString `json:"node_ids"`
	UserNames  sql.NullString `json:"user_names"`
	GroupNames sql.NullString `json:"group_names"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

func (q *Queries) RestoreRoleBinding(ctx context.Context, arg RestoreRoleBindingParams) error {
	_, err := q.db.ExecContext(ctx, RestoreRoleBinding,
		arg.Name,
		arg.RoleName,
		arg.NodeIds,
		arg.UserNames,
		arg.GroupNames,
		arg.CreatedAt,
		arg.UpdatedAt,
	)
	return err
}

const RestoreUser = `-- name: RestoreUser :exec
INSERT INTO users (
    name,
    created_at,
    updated_at
) VALUES (?, ?, ?)
`

type RestoreUserParams struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (q *Queries) RestoreUser(ctx context.Context, arg RestoreUserParams) error {
	_, err := q.db.ExecContext(ctx, RestoreUser, arg.Name, arg.CreatedAt, arg.UpdatedAt)
	return err
}
