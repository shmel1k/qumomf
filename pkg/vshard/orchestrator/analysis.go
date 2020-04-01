package orchestrator

import (
	"fmt"

	"github.com/shmel1k/qumomf/pkg/vshard"
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
	AllMasterFollowersNotReplicating ReplicaSetState = "AllMasterFollowersNotReplicating"
)

type ReplicationAnalysis struct {
	Set                      vshard.ReplicaSet
	CountReplicas            int
	CountWorkingReplicas     int
	CountReplicatingReplicas int
	State                    ReplicaSetState
}

func (a ReplicationAnalysis) String() string {
	return fmt.Sprintf(
		"ReplicaSet '%s' analysis: [State: %s; CountReplicas: %d; CountWorkingReplicas: %d; CountReplicatingReplicas: %d]",
		a.Set.UUID, a.State, a.CountReplicas, a.CountWorkingReplicas, a.CountReplicatingReplicas,
	)
}
