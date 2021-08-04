package orchestrator

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"

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

var (
	ReplicaSetStateEnum = []ReplicaSetState{
		NoProblem,
		DeadMaster,
		DeadMasterAndFollowers,
		DeadMasterAndSomeFollowers,
		DeadMasterWithoutFollowers,
		DeadFollowers,
		AllMasterFollowersNotReplicating,
		NetworkProblems,
		MasterMasterReplication,
		InconsistentVShardConfiguration,
	}
)

type ReplicationAnalysis struct {
	Set                         vshard.ReplicaSet
	CountReplicas               int // Total number of replicas in set
	CountWorkingReplicas        int // Total number of successfully discovered replicas
	CountReplicatingReplicas    int // Total number of replicas confirmed replication
	CountInconsistentVShardConf int // Total number of replicas with other than master vshard configuration
	State                       ReplicaSetState
	// DeadFollowers is a list with followers that are not currently connected to leader.
	DeadFollowers []string
}

func (a ReplicationAnalysis) String() string {
	return fmt.Sprintf(
		"[State: %s; CountReplicas: %d; CountWorkingReplicas: %d; CountReplicatingReplicas: %d]",
		a.State, a.CountReplicas, a.CountWorkingReplicas, a.CountReplicatingReplicas,
	)
}

func (a ReplicationAnalysis) GetHash() (string, error) {
	h := sha256.New()

	for _, val := range []string{
		string(a.State),
		strconv.Itoa(a.CountReplicas),
		strconv.Itoa(a.CountWorkingReplicas),
		strconv.Itoa(a.CountReplicatingReplicas),
		strconv.Itoa(a.CountInconsistentVShardConf),
		a.Set.String(),
	} {
		_, err := h.Write([]byte(val))
		if err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
