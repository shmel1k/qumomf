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
			Upstream: &vshard.Upstream{
				Status: vshard.UpstreamFollow,
				Idle:   0,
			},
			StorageInfo: vshard.StorageInfo{
				Replication: vshard.Replication{
					Status: "",
				},
				Alerts: nil,
			},
		},
		{
			UUID:           "replica_2",
			LastCheckValid: true,
			Upstream: &vshard.Upstream{
				Status: vshard.UpstreamFollow,
				Idle:   0.032492704689502716,
			},
			StorageInfo: vshard.StorageInfo{
				Replication: vshard.Replication{
					Status: vshard.StatusDisconnected,
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
			Upstream: &vshard.Upstream{
				Status: vshard.UpstreamFollow,
				Idle:   3.479430440813303,
			},
			StorageInfo: vshard.StorageInfo{
				Replication: vshard.Replication{
					Status: vshard.StatusDisconnected,
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
			Upstream: &vshard.Upstream{
				Status: vshard.UpstreamFollow,
				Idle:   0.079430440813303,
			},
			StorageInfo: vshard.StorageInfo{
				Replication: vshard.Replication{
					Status: vshard.StatusFollow,
				},
				Alerts: nil,
			},
		},
		{
			UUID:           "replica_5",
			LastCheckValid: true,
			Upstream: &vshard.Upstream{
				Status: vshard.UpstreamFollow,
				Idle:   0,
			},
			StorageInfo: vshard.StorageInfo{
				Replication: vshard.Replication{
					Status: vshard.StatusMaster,
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
