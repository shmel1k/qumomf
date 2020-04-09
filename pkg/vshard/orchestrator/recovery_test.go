package orchestrator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/shmel1k/qumomf/pkg/util"
	"github.com/shmel1k/qumomf/pkg/vshard"
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
	r := NewRecovery(mockAnalysis)

	assert.Equal(t, mockAnalysis.Set.UUID, r.SetUUID)
	assert.Equal(t, mockAnalysis.Set.MasterUUID, r.FailedUUID)
	assert.Equal(t, string(DeadMaster), r.Type)
	assert.InDelta(t, util.Timestamp(), r.StartTimestamp, 5)
}

func TestNewBlockedRecovery(t *testing.T) {
	r := NewRecovery(mockAnalysis)
	ttl := 100 * time.Second

	blocker := NewBlockedRecovery(r, ttl)
	assert.InDelta(t, time.Now().Add(ttl).UTC().Unix(), blocker.Expiration, 1)
}

func TestBlockedRecovery_Expired(t *testing.T) {
	r := NewRecovery(mockAnalysis)
	ttl := 1 * time.Second

	blocker := NewBlockedRecovery(r, ttl)
	assert.False(t, blocker.Expired())
	time.Sleep(2 * ttl)
	assert.True(t, blocker.Expired())
}
