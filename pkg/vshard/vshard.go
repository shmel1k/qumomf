package vshard

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
