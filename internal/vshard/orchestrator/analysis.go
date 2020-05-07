package orchestrator

import (
	"fmt"

	"github.com/shmel1k/qumomf/internal/vshard"
)

type AnalysisWriteStream chan<- *ReplicationAnalysis
type AnalysisReadStream <-chan *ReplicationAnalysis

func NewAnalysisStream() chan *ReplicationAnalysis {
	return make(chan *ReplicationAnalysis)
}

type ReplicaSetState string

const (
	NoProblem                        ReplicaSetState = "NoProblem"
	DeadMaster                       ReplicaSetState = "DeadMaster"
	DeadMasterAndFollowers           ReplicaSetState = "DeadMasterAndFollowers"
	DeadMasterAndSomeFollowers       ReplicaSetState = "DeadMasterAndSomeFollowers"
	DeadMasterWithoutFollowers       ReplicaSetState = "DeadMasterWithoutFollowers"
	DeadFollowers                    ReplicaSetState = "DeadFollowers"
	AllMasterFollowersNotReplicating ReplicaSetState = "AllMasterFollowersNotReplicating"
	NetworkProblems                  ReplicaSetState = "NetworkProblems"
	MasterMasterReplication          ReplicaSetState = "MasterMasterReplication"
	InconsistentVShardConfiguration  ReplicaSetState = "InconsistentVShardConfiguration"
)

type ReplicationAnalysis struct {
	Set                         vshard.ReplicaSet
	CountReplicas               int // Total number of replicas in set
	CountWorkingReplicas        int // Total number of successfully discovered replicas
	CountReplicatingReplicas    int // Total number of replicas confirmed replication
	CountInconsistentVShardConf int // Total number of replicas with other than master vshard configuration
	State                       ReplicaSetState
}

func (a ReplicationAnalysis) String() string {
	return fmt.Sprintf(
		"[State: %s; CountReplicas: %d; CountWorkingReplicas: %d; CountReplicatingReplicas: %d]",
		a.State, a.CountReplicas, a.CountWorkingReplicas, a.CountReplicatingReplicas,
	)
}
