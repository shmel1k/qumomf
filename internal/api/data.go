package api

import "github.com/shmel1k/qumomf/internal/vshard"

type ClusterInfo struct {
	Name         string `json:"name"`
	ShardsCount  int    `json:"shards_count"`
	RoutersCount int    `json:"routers_count"`
}

type AlertInfo struct {
	ClusterName string                `json:"cluster_name"`
	ShardUUID   vshard.ReplicaSetUUID `json:"shard_uuid"`
	InstanceURI string                `json:"instance_uri"`
	Alerts      []vshard.Alert        `json:"alerts"`
}
