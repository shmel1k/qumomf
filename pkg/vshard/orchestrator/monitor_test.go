package orchestrator

import (
	"fmt"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shmel1k/qumomf/pkg/vshard"
)

func Test_storageMonitor_analyze(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name string
		set  vshard.ReplicaSet
		want *ReplicationAnalysis
	}{
		{
			name: "NoProblem",
			set: vshard.ReplicaSet{
				UUID:       "set_1",
				MasterUUID: "replica_1",
				Instances: []vshard.Instance{
					mockInstance(1, true, vshard.StatusMaster),
					mockInstance(2, true, vshard.StatusFollow),
					mockInstance(3, true, vshard.StatusFollow),
				},
			},
			want: &ReplicationAnalysis{
				CountReplicas:            2,
				CountWorkingReplicas:     2,
				CountReplicatingReplicas: 2,
				State:                    NoProblem,
			},
		},
		{
			name: "NoProblem_MasterMasterReplication",
			set: vshard.ReplicaSet{
				UUID:       "set_1",
				MasterUUID: "replica_1",
				Instances: []vshard.Instance{
					mockInstance(1, true, vshard.StatusMaster),
					mockInstance(2, true, vshard.StatusMaster),
					mockInstance(3, true, vshard.StatusFollow),
				},
			},
			want: &ReplicationAnalysis{
				CountReplicas:            2,
				CountWorkingReplicas:     2,
				CountReplicatingReplicas: 2,
				State:                    NoProblem,
			},
		},
		{
			name: "DeadMaster",
			set: vshard.ReplicaSet{
				UUID:       "set_1",
				MasterUUID: "replica_1",
				Instances: []vshard.Instance{
					mockInstance(1, false, vshard.StatusMaster),
					mockInstance(2, true, vshard.StatusDisconnected),
					mockInstance(3, true, vshard.StatusDisconnected),
				},
			},
			want: &ReplicationAnalysis{
				CountReplicas:            2,
				CountWorkingReplicas:     2,
				CountReplicatingReplicas: 0,
				State:                    DeadMaster,
			},
		},
		{
			name: "DeadMaster",
			set: vshard.ReplicaSet{
				UUID:       "set_1",
				MasterUUID: "replica_1",
				Instances: []vshard.Instance{
					mockInstance(1, false, vshard.StatusMaster),
					mockInstance(2, true, vshard.StatusDisconnected),
					mockInstance(3, true, vshard.StatusDisconnected),
				},
			},
			want: &ReplicationAnalysis{
				CountReplicas:            2,
				CountWorkingReplicas:     2,
				CountReplicatingReplicas: 0,
				State:                    DeadMaster,
			},
		},
		{
			name: "DeadMasterAndFollowers",
			set: vshard.ReplicaSet{
				UUID:       "set_1",
				MasterUUID: "replica_1",
				Instances: []vshard.Instance{
					mockInstance(1, false, vshard.StatusMaster),
					mockInstance(2, false, vshard.StatusDisconnected),
					mockInstance(3, false, vshard.StatusDisconnected),
				},
			},
			want: &ReplicationAnalysis{
				CountReplicas:            2,
				CountWorkingReplicas:     0,
				CountReplicatingReplicas: 0,
				State:                    DeadMasterAndFollowers,
			},
		},
		{
			name: "DeadMasterAndSomeFollowers",
			set: vshard.ReplicaSet{
				UUID:       "set_1",
				MasterUUID: "replica_1",
				Instances: []vshard.Instance{
					mockInstance(1, false, vshard.StatusMaster),
					mockInstance(2, false, vshard.StatusDisconnected),
					mockInstance(3, true, vshard.StatusDisconnected),
				},
			},
			want: &ReplicationAnalysis{
				CountReplicas:            2,
				CountWorkingReplicas:     1,
				CountReplicatingReplicas: 0,
				State:                    DeadMasterAndSomeFollowers,
			},
		},
		{
			name: "DeadMasterWithoutFollowers",
			set: vshard.ReplicaSet{
				UUID:       "set_1",
				MasterUUID: "replica_1",
				Instances: []vshard.Instance{
					mockInstance(1, false, vshard.StatusMaster),
				},
			},
			want: &ReplicationAnalysis{
				CountReplicas:            0,
				CountWorkingReplicas:     0,
				CountReplicatingReplicas: 0,
				State:                    DeadMasterWithoutFollowers,
			},
		},
		{
			name: "AllMasterFollowersNotReplicating",
			set: vshard.ReplicaSet{
				UUID:       "set_1",
				MasterUUID: "replica_1",
				Instances: []vshard.Instance{
					mockInstance(1, true, vshard.StatusMaster),
					mockInstance(2, false, vshard.StatusFollow),
					mockInstance(3, true, vshard.StatusDisconnected),
				},
			},
			want: &ReplicationAnalysis{
				CountReplicas:            2,
				CountWorkingReplicas:     1,
				CountReplicatingReplicas: 0,
				State:                    AllMasterFollowersNotReplicating,
			},
		},
		{
			name: "NetworkProblems",
			set: vshard.ReplicaSet{
				UUID:       "set_1",
				MasterUUID: "replica_1",
				Instances: []vshard.Instance{
					mockInstance(1, false, vshard.StatusMaster),
					mockInstance(2, true, vshard.StatusFollow),
					mockInstance(3, true, vshard.StatusFollow),
				},
			},
			want: &ReplicationAnalysis{
				CountReplicas:            2,
				CountWorkingReplicas:     2,
				CountReplicatingReplicas: 2,
				State:                    NetworkProblems,
			},
		},
		{
			name: "InconsistentVShardConfiguration",
			set: vshard.ReplicaSet{
				UUID:       "set_1",
				MasterUUID: "replica_1",
				Instances: []vshard.Instance{
					mockInstance(1, true, vshard.StatusMaster),
					mockInstance(2, true, vshard.StatusFollow),
					mockInvalidVShardConf(mockInstance(3, true, vshard.StatusFollow)),
				},
			},
			want: &ReplicationAnalysis{
				CountReplicas:               2,
				CountWorkingReplicas:        2,
				CountReplicatingReplicas:    2,
				CountInconsistentVShardConf: 1,
				State:                       InconsistentVShardConfiguration,
			},
		},
	}

	for _, tv := range tests {
		tt := tv
		t.Run(tt.name, func(t *testing.T) {
			got := analyze(tt.set, logger)
			require.NotNil(t, got)
			assert.Equal(t, tt.want.CountReplicas, got.CountReplicas)
			assert.Equal(t, tt.want.CountWorkingReplicas, got.CountWorkingReplicas)
			assert.Equal(t, tt.want.CountReplicatingReplicas, got.CountReplicatingReplicas)
			assert.Equal(t, tt.want.State, got.State)
		})
	}
}

func mockInstance(id int, valid bool, status vshard.ReplicationStatus) vshard.Instance {
	return vshard.Instance{
		UUID:           vshard.InstanceUUID(fmt.Sprintf("replica_%d", id)),
		URI:            fmt.Sprintf("qumomf@replica_%d:3306", id),
		LastCheckValid: valid,
		StorageInfo: vshard.StorageInfo{
			Replication: vshard.Replication{
				Status: status,
			},
		},
	}
}

func mockInvalidVShardConf(inst vshard.Instance) vshard.Instance {
	inst.VShardFingerprint = 1000
	return inst
}
