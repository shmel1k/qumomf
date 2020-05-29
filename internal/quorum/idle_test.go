package quorum

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shmel1k/qumomf/internal/vshard"
)

func TestIdleElector(t *testing.T) {
	var testData = []struct {
		name         string
		set          vshard.ReplicaSet
		expectedUUID vshard.InstanceUUID
		expectedErr  error
	}{
		{
			name: "ShouldSelectExpectedReplica",
			set: vshard.ReplicaSet{
				Instances: []vshard.Instance{
					{
						UUID:           "1",
						LastCheckValid: false,
						StorageInfo: vshard.StorageInfo{
							Replication: vshard.Replication{
								Status: vshard.StatusMaster,
							},
						},
					},
					{
						UUID:           "2",
						LastCheckValid: true,
						Upstream: &vshard.Upstream{
							Status: vshard.UpstreamFollow,
							Idle:   0.05,
						},
						Downstream: &vshard.Downstream{
							Status: vshard.DownstreamFollow,
						},
						StorageInfo: vshard.StorageInfo{
							Replication: vshard.Replication{
								Status: vshard.StatusFollow,
							},
						},
					},
					{
						UUID:           "3",
						LastCheckValid: true,
						Upstream: &vshard.Upstream{
							Status: vshard.UpstreamFollow,
							Idle:   0.1,
						},
						Downstream: &vshard.Downstream{
							Status: vshard.DownstreamFollow,
						},
						StorageInfo: vshard.StorageInfo{
							Replication: vshard.Replication{
								Status: vshard.StatusFollow,
							},
						},
					},
				},
			},
			expectedUUID: "2",
		},
		{
			name: "NoAliveFollowers_ShouldReturnErr",
			set: vshard.ReplicaSet{
				Instances: []vshard.Instance{
					{
						UUID:           "1",
						LastCheckValid: false,
						StorageInfo: vshard.StorageInfo{
							Replication: vshard.Replication{
								Status: vshard.StatusMaster,
							},
						},
					},
					{
						UUID:           "2",
						LastCheckValid: true,
						Upstream: &vshard.Upstream{
							Status: vshard.UpstreamDisconnected,
						},
					},
					{ // too far from the master
						UUID:            "3",
						LastCheckValid:  true,
						LSNBehindMaster: 1000,
						Upstream: &vshard.Upstream{
							Status: vshard.UpstreamFollow,
							Idle:   0.1,
						},
						Downstream: &vshard.Downstream{
							Status: vshard.DownstreamFollow,
						},
						StorageInfo: vshard.StorageInfo{
							Replication: vshard.Replication{
								Status: vshard.StatusFollow,
							},
						},
					},
					{ // too far from the master
						UUID:            "4",
						LastCheckValid:  true,
						LSNBehindMaster: 1,
						Upstream: &vshard.Upstream{
							Status: vshard.UpstreamFollow,
							Idle:   10,
						},
						Downstream: &vshard.Downstream{
							Status: vshard.DownstreamFollow,
						},
						StorageInfo: vshard.StorageInfo{
							Replication: vshard.Replication{
								Status: vshard.StatusFollow,
							},
						},
					},
				},
			},
			expectedErr: ErrNoAliveFollowers,
		},
		{
			name: "EmptySet_ShouldReturnErr",
			set: vshard.ReplicaSet{
				Instances: nil,
			},
			expectedErr: ErrNoAliveFollowers,
		},
	}

	e := NewIdleElector(Options{
		ReasonableFollowerLSNLag: 100,
		ReasonableFollowerIdle:   5,
	})

	for _, v := range testData {
		vt := v
		t.Run(v.name, func(t *testing.T) {
			uuid, err := e.ChooseMaster(vt.set)
			assert.Equal(t, vt.expectedErr, err)
			assert.Equal(t, vt.expectedUUID, uuid)
		})
	}
}
