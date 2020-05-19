package orchestrator

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/shmel1k/qumomf/internal/util"
	"github.com/shmel1k/qumomf/internal/vshard"
)

// recoveryTimeFormat is a datetime format used in logs.
const recoveryTimeFormat = time.RFC3339

// RecoveryFunc is a function executed by orchestrator in case of failover.
// Returns list of recoveries applied on cluster, replica set or instances.
type RecoveryFunc func(ctx context.Context, analysis *ReplicationAnalysis) []*Recovery

type RecoveryScope string

const (
	RecoveryScopeInstance RecoveryScope = "instance"
	RecoveryScopeSet      RecoveryScope = "replica set"
)

// Recovery describes the applied recovery to a cluster, replica set or instance.
type Recovery struct {
	Type           string
	Scope          RecoveryScope
	AnalysisEntry  ReplicationAnalysis
	ClusterName    string
	SetUUID        vshard.ReplicaSetUUID
	Failed         vshard.InstanceIdent
	Successor      vshard.InstanceIdent
	IsSuccessful   bool
	StartTimestamp int64
	EndTimestamp   int64
	Expiration     int64
}

func NewRecovery(scope RecoveryScope, failed vshard.InstanceIdent, analysis ReplicationAnalysis) *Recovery {
	return &Recovery{
		Type:           string(analysis.State),
		Scope:          scope,
		AnalysisEntry:  analysis,
		SetUUID:        analysis.Set.UUID,
		Failed:         failed,
		StartTimestamp: util.Timestamp(),
	}
}

func (r *Recovery) ExpireAfter(ttl time.Duration) {
	exp := time.Now().Add(ttl).Unix()
	r.Expiration = exp
}

// ScopeKey returns the UUID of the replica set or instance
// where recovery has been applied on.
func (r *Recovery) ScopeKey() string {
	switch r.Scope {
	case RecoveryScopeInstance:
		return string(r.Failed.UUID)
	case RecoveryScopeSet:
		return string(r.SetUUID)
	}

	return r.ClusterName
}

func (r *Recovery) Expired() bool {
	now := util.Timestamp()
	return r.Expiration < now
}

func (r *Recovery) String() string {
	start := time.Unix(r.StartTimestamp, 0).Format(recoveryTimeFormat)
	end := time.Unix(r.EndTimestamp, 0).Format(recoveryTimeFormat)
	duration := r.EndTimestamp - r.StartTimestamp

	var sb strings.Builder
	sb.WriteString("set: ")
	sb.WriteString(string(r.SetUUID))
	sb.WriteString(", type: ")
	sb.WriteString(r.Type)
	sb.WriteString(", failed: ")
	sb.WriteString(string(r.Failed.UUID))
	if r.Successor.UUID != "" {
		sb.WriteString(", successor: ")
		sb.WriteString(string(r.Successor.UUID))
	}
	sb.WriteString(", success: ")
	sb.WriteString(strconv.FormatBool(r.IsSuccessful))
	sb.WriteString(", period: ")
	sb.WriteString(start)
	sb.WriteString(" - ")
	sb.WriteString(end)
	sb.WriteString(", duration: ")
	sb.WriteString(strconv.FormatInt(duration, 10))
	sb.WriteString("s")

	return sb.String()
}
