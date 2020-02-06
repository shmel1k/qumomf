package quorum

import (
	"testing"

	"github.com/shmel1k/qumomf/pkg/vshard"
)

func TestLagQuorum(t *testing.T) {
	var testData = []struct {
		info         vshard.ShardInfo
		testName     string
		expectedUUID string
		expectedErr  error
	}{
		{
			info: vshard.ShardInfo{
				vshard.ReplicaInfo{
					Lag:    0.01,
					Status: vshard.StatusFollow,
					UUID:   "1",
				},
				vshard.ReplicaInfo{
					Lag:    0.05,
					Status: vshard.StatusFollow,
					UUID:   "2",
				},
			},
			expectedUUID: "1",
			testName:     "ok",
		},
		{
			info: vshard.ShardInfo{
				vshard.ReplicaInfo{
					Status: vshard.StatusMaster,
					UUID:   "1",
				},
			},
			expectedErr: ErrNoReplicaFound,
			testName:    "only master",
		},
		{
			info:        nil,
			expectedErr: ErrEmptyInfo,
			testName:    "empty info",
		},
	}

	l := &lagQuorum{}

	for _, v := range testData {
		t.Run(v.testName, func(t *testing.T) {
			uid, err := l.ChooseMaster(v.info)
			if err != v.expectedErr {
				t.Errorf("got err %v, expected %v", err, v.expectedErr)
			}
			if uid != v.expectedUUID {
				t.Errorf("got uid %q, got %q", uid, v.expectedUUID)
			}
		})
	}
}
