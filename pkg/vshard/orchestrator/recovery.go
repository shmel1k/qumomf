package orchestrator

import (
	"fmt"
	"time"

	"github.com/shmel1k/qumomf/pkg/util"
	"github.com/shmel1k/qumomf/pkg/vshard"
)

type Recovery struct {
	SetUUID        vshard.ReplicaSetUUID
	FailedUUID     vshard.InstanceUUID
	SuccessorUUID  vshard.InstanceUUID
	Type           string
	IsSuccessful   bool
	StartTimestamp int64
	EndTimestamp   int64
}

func NewRecovery(analysis *ReplicationAnalysis) *Recovery {
	return &Recovery{
		SetUUID:        analysis.Set.UUID,
		FailedUUID:     analysis.Set.MasterUUID,
		Type:           string(analysis.State),
		StartTimestamp: util.Timestamp(),
	}
}

func (r Recovery) String() string {
	start := time.Unix(r.StartTimestamp, 0).Format(time.RFC3339)
	end := time.Unix(r.EndTimestamp, 0).Format(time.RFC3339)
	duration := r.EndTimestamp - r.StartTimestamp

	return fmt.Sprintf("set: %s, type: %s, failed: %s, successor: %s, success: %t, period: %s - %s, duration: %ds", r.SetUUID, r.Type, r.FailedUUID, r.SuccessorUUID, r.IsSuccessful, start, end, duration)
}

type BlockedRecovery struct {
	Recovery   *Recovery
	Expiration int64
}

func NewBlockedRecovery(r *Recovery, ttl time.Duration) *BlockedRecovery {
	exp := time.Now().Add(ttl).UTC().Unix()
	return &BlockedRecovery{
		Recovery:   r,
		Expiration: exp,
	}
}

func (b BlockedRecovery) Expired() bool {
	now := util.Timestamp()
	return b.Expiration < now
}
