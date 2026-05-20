package repository

import (
	"context"
	"matrix/api/domain"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

type Neo4jRepo struct {
	Driver neo4j.Driver
	DBName string
}

func NewNeo4jRepo(driver neo4j.Driver, dbName string) *Neo4jRepo {
	return &Neo4jRepo{
		Driver: driver,
		DBName: dbName,
	}
}

func (r *Neo4jRepo) FindDeviceByID(ctx context.Context, id string) (*domain.Device, error) {
	query := `MATCH (d:Device {id: $id}) RETURN d`
	result, err := neo4j.ExecuteQuery(ctx, r.Driver, query, map[string]any{"id": id},
		neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(r.DBName),
	)
	if err != nil {
		return nil, err
	}
	if len(result.Records) == 0 {
		return nil, nil
	}
	node, _, _ := neo4j.GetRecordValue[neo4j.Node](result.Records[0], "d")
	dev := nodeToDevice(node)
	return dev, nil
}

func (r *Neo4jRepo) FindDevices(ctx context.Context, first int, after string) ([]*domain.Device, bool, string, error) {
	query := `MATCH (d:Device)
	WHERE $after = "" OR d.id > $after
    RETURN d
    ORDER BY d.id
    LIMIT $limit
	`
	result, err := neo4j.ExecuteQuery(ctx, r.Driver, query, map[string]any{
		"after": after,
		"limit": first + 1,
	}, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(r.DBName))
	if err != nil {
		return nil, false, "", err
	}

	var devices []*domain.Device
	for _, record := range result.Records {
		node, _, _ := neo4j.GetRecordValue[neo4j.Node](record, "d")
		devices = append(devices, nodeToDevice(node))
	}

	hasNextPage := len(devices) > first
	var endCursor string
	if hasNextPage {
		devices = devices[:first]
	}
	if len(devices) > 0 {
		endCursor = devices[len(devices)-1].ID
	}
	return devices, hasNextPage, endCursor, nil
}

func (r *Neo4jRepo) BatchIPsByDeviceIDs(ctx context.Context, deviceIDs []string, limit int) (map[string][]*domain.IPv4Addr, error) {
	query := `MATCH (d:Device)-[:HAS_IP]->(ip:IPAddress)
    WHERE d.id IN $deviceIDs
    RETURN d.id AS deviceID, collect(ip)[0..$limit] AS ips
    `
	params := map[string]any{
		"deviceIDs": deviceIDs,
		"limit":     limit,
	}
	result, err := neo4j.ExecuteQuery(ctx, r.Driver, query, params, neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.DBName))
	if err != nil {
		return nil, err
	}
	out := make(map[string][]*domain.IPv4Addr, len(deviceIDs))
	for _, id := range deviceIDs {
		out[id] = []*domain.IPv4Addr{}
	}

	for _, record := range result.Records {
		devID, _, _ := neo4j.GetRecordValue[string](record, "deviceID")
		values, _, _ := neo4j.GetRecordValue[[]any](record, "ips")
		for _, value := range values {
			node := value.(neo4j.Node)
			out[devID] = append(out[devID], nodeToIPAddress(node))
		}
	}

	return out, nil
}

func (r *Neo4jRepo) BatchConnectionsByDeviceIDs(ctx context.Context, deviceIDs []string, limit int) (map[string][]*domain.DeviceLink, error) {
	query := `MATCH (d:Device)-[rel:CONNECTED_TO]->(target:Device)
    WHERE d.id IN $deviceIDS
    RETURN d.id AS deviceID, collect({target: target, rel: rel})[0..$limit] AS links`
	params := map[string]any{
		"deviceIDS": deviceIDs,
		"limit":     limit,
	}
	result, err := neo4j.ExecuteQuery(ctx, r.Driver, query, params, neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.DBName))
	if err != nil {
		return nil, err
	}
	out := make(map[string][]*domain.DeviceLink, len(deviceIDs))
	for _, id := range deviceIDs {
		out[id] = []*domain.DeviceLink{}
	}
	for _, record := range result.Records {
		deviceID, _, _ := neo4j.GetRecordValue[string](record, "deviceID")
		values, _, _ := neo4j.GetRecordValue[[]any](record, "links")
		for _, value := range values {
			item := value.(map[string]any)
			targetNode := item["target"].(neo4j.Node)
			rel := item["rel"].(neo4j.Relationship)
			out[deviceID] = append(out[deviceID], &domain.DeviceLink{
				Target: nodeToDevice(targetNode),
				Relation: &domain.GraphRelation{
					ID:    rel.ElementId,
					Type:  rel.Type,
					Props: rel.Props,
				},
			})
		}
	}
	return out, nil
}

func nodeToDevice(node neo4j.Node) *domain.Device {
	get := func(key string) string {
		if v, ok := node.Props[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	return &domain.Device{
		ID:          get("id"),
		Name:        get("name"),
		Type:        get("type"),
		MIP:         get("mip"),
		LoginUser:   get("login_user"),
		LoginMethod: get("login_method"),
		LoginPasswd: get("login_passwd"),
	}
}

func nodeToIPAddress(node neo4j.Node) *domain.IPv4Addr {
	get := func(key string) string {
		if v, ok := node.Props[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	getUint64 := func(key string) uint64 {
		if v, ok := node.Props[key]; ok {
			if n, ok := v.(int64); ok {
				return uint64(n)
			}
		}
		return 0
	}
	return &domain.IPv4Addr{
		ID:        get("id"),
		IP:        get("address"),
		Mask:      get("mask"),
		Cidr:      get("cidr"),
		StartAddr: getUint64("start_addr"),
		EndAddr:   getUint64("end_addr"),
	}
}
