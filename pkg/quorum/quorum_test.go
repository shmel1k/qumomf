package quorum

import (
	"testing"

	"github.com/shmel1k/qumomf/pkg/vshard"
)

func TestLagQuorum(t *testing.T) {
	var testData = []struct {
		testName     string
		set          vshard.ReplicaSet
		expectedUUID vshard.InstanceUUID
		expectedErr  error
	}{
		{
			testName: "ShouldSelectExpectedReplica",
			set: vshard.ReplicaSet{
				Instances: []vshard.Instance{
					{
						UUID: "1",
						StorageInfo: vshard.StorageInfo{
							ReplicationStatus: vshard.StatusMaster,
						},
					},
					{
						UUID: "2",
						Upstream: &vshard.Upstream{
							Status: vshard.UpstreamFollow,
							Lag:    0.05,
						},
						StorageInfo: vshard.StorageInfo{
							ReplicationStatus: vshard.StatusFollow,
						},
					},
					{
						UUID: "3",
						Upstream: &vshard.Upstream{
							Status: vshard.UpstreamFollow,
							Lag:    0.1,
						},
						StorageInfo: vshard.StorageInfo{
							ReplicationStatus: vshard.StatusFollow,
						},
					},
				},
			},
			expectedUUID: "2",
		},
		{
			testName: "NoFollowers_ShouldReturnErr",
			set: vshard.ReplicaSet{
				Instances: []vshard.Instance{
					{
						UUID: "1",
						StorageInfo: vshard.StorageInfo{
							ReplicationStatus: vshard.StatusMaster,
						},
					},
				},
			},
			expectedErr: ErrNoFollowers,
		},
		{
			testName: "EmptySet_ShouldReturnErr",
			set: vshard.ReplicaSet{
				Instances: nil,
			},
			expectedErr: ErrNoFollowers,
		},
	}

	l := &lagQuorum{}

	for _, v := range testData {
		vt := v
		t.Run(v.testName, func(t *testing.T) {
			uid, err := l.ChooseMaster(vt.set)
			if err != vt.expectedErr {
				t.Errorf("got err %v, expected %v", err, vt.expectedErr)
			}
			if uid != vt.expectedUUID {
				t.Errorf("got uid %q, got %q", uid, vt.expectedUUID)
			}
		})
	}
}
