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
	ttl := 100 * time.Second
	r := NewSetRecovery(mockAnalysis, ttl)

	assert.Equal(t, mockAnalysis.Set.UUID, r.SetUUID)
	assert.Equal(t, mockAnalysis.Set.MasterUUID, r.FailedUUID)
	assert.Equal(t, string(DeadMaster), r.Reason())
	assert.InDelta(t, util.Timestamp(), r.StartTimestamp, 5)
	assert.InDelta(t, time.Now().Add(ttl).UTC().Unix(), r.Expiration, 1)
}

func TestRecovery_Expired(t *testing.T) {
	ttl := 1 * time.Second
	r := NewSetRecovery(mockAnalysis, ttl)

	assert.False(t, r.Expired())
	time.Sleep(2 * ttl)
	assert.True(t, r.Expired())
}
