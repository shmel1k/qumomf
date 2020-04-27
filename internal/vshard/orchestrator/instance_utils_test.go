package orchestrator

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shmel1k/qumomf/internal/vshard"
)

func TestInstanceFailoverSorter(t *testing.T) {
	instances := []vshard.Instance{
		{
			UUID:           "replica_1",
			LastCheckValid: false,
			StorageInfo: vshard.StorageInfo{
				Replication: vshard.Replication{
					Status: "",
					Delay:  0,
				},
				Alerts: nil,
			},
		},
		{
			UUID:           "replica_2",
			LastCheckValid: true,
			StorageInfo: vshard.StorageInfo{
				Replication: vshard.Replication{
					Status: vshard.StatusDisconnected,
					Delay:  0.032492704689502716,
				},
				Alerts: []vshard.Alert{
					{
						Type:        vshard.AlertUnreachableMaster,
						Description: "Master of replicaset is unreachable: disconnected",
					},
				},
			},
		},
		{
			UUID:           "replica_3",
			LastCheckValid: true,
			StorageInfo: vshard.StorageInfo{
				Replication: vshard.Replication{
					Status: vshard.StatusDisconnected,
					Delay:  3.479430440813303,
				},
				Alerts: []vshard.Alert{
					{
						Type:        vshard.AlertUnreachableMaster,
						Description: "Master of replicaset is unreachable: disconnected",
					},
				},
			},
		},
		{
			UUID:           "replica_4",
			LastCheckValid: true,
			StorageInfo: vshard.StorageInfo{
				Replication: vshard.Replication{
					Status: vshard.StatusFollow,
					Delay:  0.079430440813303,
				},
				Alerts: nil,
			},
		},
		{
			UUID:           "replica_5",
			LastCheckValid: true,
			StorageInfo: vshard.StorageInfo{
				Replication: vshard.Replication{
					Status: vshard.StatusMaster,
					Delay:  0,
				},
				Alerts: nil,
			},
		},
	}

	sort.Sort(NewInstanceFailoverSorter(instances))

	expected := []vshard.InstanceUUID{
		"replica_2", "replica_3", "replica_5", "replica_4", "replica_1",
	}

	got := make([]vshard.InstanceUUID, 0, len(instances))
	for _, inst := range instances {
		got = append(got, inst.UUID)
	}

	assert.Equal(t, expected, got)
}
