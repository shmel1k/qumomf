package quorum

import (
	"testing"

	"github.com/shmel1k/qumomf/pkg/vshard"
)

func TestLagQuorum(t *testing.T) {
	var testData = []struct {
		info         vshard.ReplicaSetInfo
		testName     string
		expectedUUID vshard.ReplicaUUID
		expectedErr  error
	}{
		{
			info: vshard.ReplicaSetInfo{
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
			info: vshard.ReplicaSetInfo{
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
		vt := v
		t.Run(v.testName, func(t *testing.T) {
			uid, err := l.ChooseMaster(vt.info)
			if err != vt.expectedErr {
				t.Errorf("got err %v, expected %v", err, vt.expectedErr)
			}
			if uid != vt.expectedUUID {
				t.Errorf("got uid %q, got %q", uid, vt.expectedUUID)
			}
		})
	}
}
