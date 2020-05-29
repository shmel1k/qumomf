package quorum

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shmel1k/qumomf/internal/vshard"
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
						Priority: 100,
					},
					{ // good candidate but has lower priority
						UUID:              "3",
						LastCheckValid:    true,
						LSNBehindMaster:   0,
						VShardFingerprint: 100,
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
						Priority: 10,
					},
					{ // too far from master
						UUID:              "4",
						LastCheckValid:    true,
						LSNBehindMaster:   10,
						VShardFingerprint: 100,
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
					{ // inconsistent vshard configuration
						UUID:              "5",
						LastCheckValid:    true,
						LSNBehindMaster:   0,
						VShardFingerprint: 10,
						Upstream: &vshard.Upstream{
							Status: vshard.UpstreamFollow,
							Idle:   0.0001,
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

	e := NewSmartElector(Options{
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

func Test_inDelta(t *testing.T) {
	tests := []struct {
		name  string
		d1    float64
		d2    float64
		delta float64
		want  bool
	}{
		{
			name:  "InDelta",
			d1:    0.23,
			d2:    0.532,
			delta: 1,
			want:  true,
		},
		{
			name:  "NotInDelta",
			d1:    0.23,
			d2:    0.532,
			delta: 0.1,
			want:  false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, inDelta(tt.d1, tt.d2, tt.delta))
		})
	}
}
