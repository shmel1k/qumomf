package quorum

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shmel1k/qumomf/pkg/vshard"
)

func Test_smartElector_ChooseMaster(t *testing.T) {
	var testData = []struct {
		name         string
		set          vshard.ReplicaSet
		expectedUUID vshard.InstanceUUID
		expectedErr  error
	}{
		{
			name: "ShouldSelectExpectedReplica",
			set: vshard.ReplicaSet{
				MasterUUID: "1",
				Instances: []vshard.Instance{
					{
						UUID:              "1",
						LastCheckValid:    false,
						VShardFingerprint: 100,
						StorageInfo: vshard.StorageInfo{
							Replication: vshard.Replication{
								Status: vshard.StatusMaster,
							},
						},
					},
					{ // the best candidate
						UUID:              "2",
						LastCheckValid:    true,
						LSNBehindMaster:   0,
						VShardFingerprint: 100,
						Upstream: &vshard.Upstream{
							Status: vshard.UpstreamFollow,
						},
						Downstream: &vshard.Downstream{
							Status: vshard.DownstreamFollow,
						},
						StorageInfo: vshard.StorageInfo{
							Replication: vshard.Replication{
								Status: vshard.StatusFollow,
								Delay:  0.05,
							},
						},
					},
					{ // too far from master
						UUID:              "3",
						LastCheckValid:    true,
						LSNBehindMaster:   10,
						VShardFingerprint: 100,
						Upstream: &vshard.Upstream{
							Status: vshard.UpstreamFollow,
						},
						Downstream: &vshard.Downstream{
							Status: vshard.DownstreamFollow,
						},
						StorageInfo: vshard.StorageInfo{
							Replication: vshard.Replication{
								Status: vshard.StatusFollow,
								Delay:  0.1,
							},
						},
					},
					{ // inconsistent vshard configuration
						UUID:              "4",
						LastCheckValid:    true,
						LSNBehindMaster:   0,
						VShardFingerprint: 10,
						Upstream: &vshard.Upstream{
							Status: vshard.UpstreamFollow,
						},
						Downstream: &vshard.Downstream{
							Status: vshard.DownstreamFollow,
						},
						StorageInfo: vshard.StorageInfo{
							Replication: vshard.Replication{
								Status: vshard.StatusFollow,
								Delay:  0.0001,
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
				MasterUUID: "1",
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

	e := NewSmartElector()

	for _, v := range testData {
		vt := v
		t.Run(v.name, func(t *testing.T) {
			uuid, err := e.ChooseMaster(vt.set)
			assert.Equal(t, vt.expectedErr, err)
			assert.Equal(t, vt.expectedUUID, uuid)
		})
	}
}
