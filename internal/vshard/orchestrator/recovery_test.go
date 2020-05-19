package orchestrator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/shmel1k/qumomf/internal/util"
	"github.com/shmel1k/qumomf/internal/vshard"
)

var mockAnalysis = &ReplicationAnalysis{
	Set: vshard.ReplicaSet{
		UUID:       "set_uuid",
		MasterUUID: "master_uuid",
	},
	CountReplicas:            3,
	CountWorkingReplicas:     0,
	CountReplicatingReplicas: 0,
	State:                    DeadMaster,
}

func TestNewRecovery(t *testing.T) {
	ttl := 100 * time.Second
	failed := vshard.InstanceIdent{
		UUID: "master",
		URI:  "localhost:3301",
	}
	r := NewRecovery(RecoveryScopeSet, failed, *mockAnalysis)
	r.ExpireAfter(ttl)

	assert.Equal(t, *mockAnalysis, r.AnalysisEntry)
	assert.Equal(t, mockAnalysis.Set.UUID, r.SetUUID)
	assert.Equal(t, failed.UUID, r.Failed.UUID)
	assert.Equal(t, failed.URI, r.Failed.URI)
	assert.Equal(t, string(DeadMaster), r.Type)
	assert.InDelta(t, util.Timestamp(), r.StartTimestamp, 5)
	assert.InDelta(t, time.Now().Add(ttl).UTC().Unix(), r.Expiration, 1)
}

func TestRecovery_Expired(t *testing.T) {
	ttl := 1 * time.Second
	failed := vshard.InstanceIdent{
		UUID: "master",
		URI:  "localhost:3301",
	}
	r := NewRecovery(RecoveryScopeInstance, failed, *mockAnalysis)
	r.ExpireAfter(ttl)

	assert.False(t, r.Expired())
	time.Sleep(2 * ttl)
	assert.True(t, r.Expired())
}
