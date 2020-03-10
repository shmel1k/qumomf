package orchestrator

import (
	"context"
	"github.com/shmel1k/qumomf/pkg/quorum"
	"github.com/shmel1k/qumomf/pkg/vshard"
	"testing"
)

var dummyContext = context.Background()

type CheckAndRecoverTest struct {
	name string
	replicaSetAnalysis ReplicaSetAnalysis
	expectedMaster string
}

func TestCheckAndRecover(t *testing.T) {
	swapMasterFailover := swapMasterFailover{
		cluster:vshard.NewCluster([]vshard.InstanceConfig{}, map[vshard.ShardUUID][]vshard.InstanceConfig{}),
		elector: quorum.NewLagQuorum(),
	}
	tests := []CheckAndRecoverTest {
		{
			name: "dead master",
			replicaSetAnalysis: ReplicaSetAnalysis{
				Set: vshard.NewReplicaSet("1", []*vshard.Connector{}),
				Info: vshard.ReplicaSetInfo{
					vshard.ReplicaInfo{UUID: "1", State: vshard.DeadMaster, Status: vshard.DeadMaster, Lag: 0.01},
					vshard.ReplicaInfo{UUID: "2", Status: vshard.StatusFollow, State: vshard.NoProblem, Lag: 0.02},
				},
			},
			expectedMaster: "2",
		},
		{
			name: "DeadSlave",
			replicaSetAnalysis: ReplicaSetAnalysis{
				Set:  vshard.NewReplicaSet("2", []*vshard.Connector{}),
				Info: vshard.ReplicaSetInfo{
					vshard.ReplicaInfo{UUID: "1", State:vshard.DeadSlave, Status:vshard.StatusFollow, Lag: 0.01},
					vshard.ReplicaInfo{UUID: "2", State:vshard.NoProblem, Status:vshard.StatusMaster, Lag: 0.02},
				},
			},
			expectedMaster: "2",
		},
		{
			name: "BadStorageInfo",
			replicaSetAnalysis: ReplicaSetAnalysis{
				Set:  vshard.NewReplicaSet("3", []*vshard.Connector{}),
				Info: vshard.ReplicaSetInfo{
					vshard.ReplicaInfo{UUID: "1", State:vshard.BadStorageInfo, Status:vshard.StatusFollow, Lag: 0.01},
					vshard.ReplicaInfo{UUID: "2", State:vshard.NoProblem, Status:vshard.StatusMaster, Lag: 0.02},
				},
			},
			expectedMaster: "2",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			swapMasterFailover.checkAndRecover(dummyContext, test.replicaSetAnalysis)
		})
	}
}

type PromoteFollowerToMasterTest struct {
	name string
}

//func TestPromoteFollowerToMaster(t *testing.T) {
//	swapMasterFailover := swapMasterFailover{
//		cluster:vshard.NewCluster([]vshard.InstanceConfig{}, map[vshard.ShardUUID][]vshard.InstanceConfig{}),
//		elector: quorum.NewLagQuorum(),
//	}
//	tests := []PromoteFollowerToMasterTest {
//		{
//			name: ,
//		},
//	}
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			swapMasterFailover.promoteFollowerToMaster(dummyContext, )
//		})
//	}
//}