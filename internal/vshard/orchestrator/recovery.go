package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/shmel1k/qumomf/internal/util"
	"github.com/shmel1k/qumomf/internal/vshard"
)

// recoveryTimeFormat is a datetime format used in logs.
const recoveryTimeFormat = time.RFC3339

// RecoveryFunc is a function executed by orchestrator in case of failover.
// Returns list of recoveries applied on cluster, replica set or instances.
type RecoveryFunc func(ctx context.Context, analysis *ReplicationAnalysis) []Recovery

// Recovery describes the applied recovery to a cluster instances.
type Recovery interface {
	Reason() string
	LockKey() string
	Expired() bool
	Succeed() bool
}

type SetRecovery struct {
	SetUUID        vshard.ReplicaSetUUID
	FailedUUID     vshard.InstanceUUID
	SuccessorUUID  vshard.InstanceUUID
	Type           string
	IsSuccessful   bool
	StartTimestamp int64
	EndTimestamp   int64
	Expiration     int64
}

func NewSetRecovery(analysis *ReplicationAnalysis, ttl time.Duration) *SetRecovery {
	exp := time.Now().Add(ttl).UTC().Unix()

	return &SetRecovery{
		SetUUID:        analysis.Set.UUID,
		FailedUUID:     analysis.Set.MasterUUID,
		Type:           string(analysis.State),
		StartTimestamp: util.Timestamp(),
		Expiration:     exp,
	}
}

func (r SetRecovery) Reason() string {
	return r.Type
}

func (r SetRecovery) LockKey() string {
	return string(r.SetUUID)
}

func (r SetRecovery) Expired() bool {
	now := util.Timestamp()
	return r.Expiration < now
}

func (r SetRecovery) Succeed() bool {
	return r.IsSuccessful
}

func (r SetRecovery) String() string {
	start := time.Unix(r.StartTimestamp, 0).Format(recoveryTimeFormat)
	end := time.Unix(r.EndTimestamp, 0).Format(recoveryTimeFormat)
	duration := r.EndTimestamp - r.StartTimestamp

	return fmt.Sprintf("set: %s, type: %s, failed: %s, successor: %s, success: %t, period: %s - %s, duration: %ds", r.SetUUID, r.Type, r.FailedUUID, r.SuccessorUUID, r.IsSuccessful, start, end, duration)
}

type InstanceRecovery struct {
	UUID           vshard.InstanceUUID
	Type           string
	IsSuccessful   bool
	StartTimestamp int64
	EndTimestamp   int64
	Expiration     int64
}

func NewInstanceRecovery(uuid vshard.InstanceUUID, reason string, ttl time.Duration) *InstanceRecovery {
	exp := time.Now().Add(ttl).UTC().Unix()

	return &InstanceRecovery{
		UUID:           uuid,
		Type:           reason,
		StartTimestamp: util.Timestamp(),
		Expiration:     exp,
	}
}

func (r InstanceRecovery) Reason() string {
	return r.Type
}

func (r InstanceRecovery) LockKey() string {
	return string(r.UUID)
}

func (r InstanceRecovery) Expired() bool {
	now := util.Timestamp()
	return r.Expiration < now
}

func (r InstanceRecovery) Succeed() bool {
	return r.IsSuccessful
}

func (r InstanceRecovery) String() string {
	start := time.Unix(r.StartTimestamp, 0).Format(recoveryTimeFormat)
	end := time.Unix(r.EndTimestamp, 0).Format(recoveryTimeFormat)
	duration := r.EndTimestamp - r.StartTimestamp

	return fmt.Sprintf("instance: %s, type: %s, success: %t, period: %s - %s, duration: %ds", r.UUID, r.Type, r.IsSuccessful, start, end, duration)
}
