package vshard

import "fmt"

type RouterUUID string
type ShardUUID string
type ReplicaUUID string
type ReplicaStatus string
type ReplicaRole string

const (
	NoProblem       ReplicaStatus = "NoProblem"
	DeadMaster                    = "DeadMaster"
	DeadSlave                     = "DeadSlave"
	BadStorageInfo                = "BadStorageInfo"
	HasActiveAlerts               = "HasActiveAlerts"
)

const (
	RoleFollow ReplicaRole = "follow"
	RoleMaster ReplicaRole = "master"
)

type ReplicaInfo struct {
	UUID   ReplicaUUID
	Role   ReplicaRole
	Status ReplicaStatus
	Lag    float64
	Alerts []interface{}
}

func (i ReplicaInfo) String() string {
	return fmt.Sprintf("UUID: %s, Role: %s, Status: %s, Lag: %f, Alerts: %v", i.UUID, i.Role, i.Status, i.Lag, i.Alerts)
}

type ReplicaSetInfo []ReplicaInfo

type ReplicaConfig struct {
	Name   string
	URI    string
	Master bool
}

type ReplicaSetConfig struct {
	Replicas map[ReplicaUUID]ReplicaConfig
}

type ShardingConfig struct {
	Shards map[ShardUUID]ReplicaSetConfig
}

type CommonConfig struct {
	Sharding    ShardingConfig
	BucketCount uint32
}

func PrepareURI(user, password, addr string) string {
	return fmt.Sprintf("%s:%s@%s", user, password, addr)
}
