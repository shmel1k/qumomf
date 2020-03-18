package vshard

import "fmt"

type RouterUUID string
type ShardUUID string
type ReplicaUUID string
type ReplicaState string
type ReplicaStatus string

const (
	NoProblem       ReplicaState = "NoProblem"
	DeadMaster      ReplicaState = "DeadMaster"
	DeadSlave       ReplicaState = "DeadSlave"
	BadStorageInfo  ReplicaState = "BadStorageInfo"
	HasActiveAlerts ReplicaState = "HasActiveAlerts"
)

const (
	StatusFollow ReplicaStatus = "follow"
	StatusMaster ReplicaStatus = "master"
)

type ReplicaInfo struct {
	UUID   ReplicaUUID
	Status ReplicaStatus
	State  ReplicaState
	Lag    float64
	Alerts []interface{}
}

func (i ReplicaInfo) String() string {
	return fmt.Sprintf("UUID: %s, Status: %s, State: %s, Lag: %f, Alerts: %v", i.UUID, i.Status, i.State, i.Lag, i.Alerts)
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
