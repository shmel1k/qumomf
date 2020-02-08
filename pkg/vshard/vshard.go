package vshard

import "fmt"

type ReplicationStatus string

var (
	StatusFollow ReplicationStatus = "follow"
	StatusMaster ReplicationStatus = "master"
)

type ReplicaInfo struct {
	Status ReplicationStatus
	Lag    float64
	UUID   string
	Alerts []string
}

type ShardInfo []ReplicaInfo

type ReplicaConfig struct {
	Name   string
	URI    string
	Master bool
}

type ReplicasetConfig struct {
	Replicas map[string]ReplicaConfig
}

type ShardingConfig struct {
	Shards map[string]ReplicasetConfig
}

type CommonConfig struct {
	Sharding    ShardingConfig
	BucketCount uint32
}

func PrepareURI(user, password, addr string) string {
	return fmt.Sprintf("%s:%s@%s", user, password, addr)
}
