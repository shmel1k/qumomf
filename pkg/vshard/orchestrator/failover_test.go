package orchestrator

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/shmel1k/qumomf/pkg/quorum"
	"github.com/shmel1k/qumomf/pkg/util"
	"github.com/shmel1k/qumomf/pkg/vshard"
)

type failoverTestSuite struct {
	suite.Suite
	cluster  *vshard.Cluster
	failover Failover
}

func (s *failoverTestSuite) SetupTest() {
	s.cluster = vshard.MockCluster()
	s.cluster.SetReadOnly(false)
}

func (s *failoverTestSuite) AfterTest(_, _ string) {
	if s.failover != nil {
		s.failover.Shutdown()
	}
	if s.cluster != nil {
		s.cluster.Shutdown()
	}
}

func (s *failoverTestSuite) Test_failover_promoteFollowerToMaster() {
	t := s.T()

	if testing.Short() {
		t.Skip("test requires dev env - skipping it in short mode.")
	}

	s.cluster.Discover()
	require.InDelta(t, util.Timestamp(), s.cluster.LastDiscovered(), 1)

	elector := quorum.NewLagQuorum()
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	s.failover = NewDefaultFailover(s.cluster, FailoverConfig{
		Elector:                     elector,
		ReplicaSetRecoveryBlockTime: 2 * time.Second,
	}, logger)
	fv := s.failover.(*failover)

	stream := NewAnalysisStream()
	fv.Serve(stream)

	set, err := s.cluster.ReplicaSet("7432f072-c00b-4498-b1a6-6d9547a8a150")
	require.Nil(t, err)

	analysis := &ReplicationAnalysis{
		Set:                      set,
		CountReplicas:            1,
		CountWorkingReplicas:     0,
		CountReplicatingReplicas: 0,
		State:                    DeadMaster,
	}
	stream <- analysis

	require.Eventually(t, func() bool {
		return fv.hasBlockedRecovery(string(set.UUID))
	}, 5*time.Second, 100*time.Millisecond)
	require.Len(t, fv.recoveries, 1)
	recv, ok := fv.recoveries[0].(*SetRecovery)
	require.True(t, ok)

	require.True(t, recv.IsSuccessful)
	assert.InDelta(t, util.Timestamp(), recv.StartTimestamp, 5)
	assert.InDelta(t, util.Timestamp(), recv.EndTimestamp, 2)
	assert.Equal(t, string(analysis.State), recv.Reason())
	assert.Equal(t, set.MasterUUID, recv.FailedUUID)

	recvSet, err := s.cluster.ReplicaSet("7432f072-c00b-4498-b1a6-6d9547a8a150")
	require.Nil(t, err)

	assert.Equal(t, recv.SuccessorUUID, recvSet.MasterUUID)

	master, err := recvSet.Master()
	require.Nil(t, err)
	assert.False(t, master.Readonly)

	alive := recvSet.AliveFollowers()
	assert.Len(t, alive, 1)
	for i := range alive {
		assert.True(t, alive[i].Readonly)
	}

	// Ensure that anti-flapping is working.
	analysis.Set = recvSet
	stream <- analysis

	require.Len(t, fv.recoveries, 1)
	assert.Same(t, recv, fv.recoveries[0])

	// Recreate the initial cluster.
	fv.cleanup(true)
	require.False(t, fv.hasBlockedRecovery(string(set.UUID)))

	stream <- analysis

	require.Eventually(t, func() bool {
		return fv.hasBlockedRecovery(string(set.UUID))
	}, 5*time.Second, 100*time.Millisecond)
	require.Len(t, fv.recoveries, 1)
	assert.True(t, recv != fv.recoveries[0])

	recv, ok = fv.recoveries[0].(*SetRecovery)
	require.True(t, ok)
	assert.True(t, recv.IsSuccessful)
	assert.Equal(t, set.MasterUUID, recv.SuccessorUUID)
}

func (s *failoverTestSuite) Test_failover_wishEventualConsistency() {
	t := s.T()

	if testing.Short() {
		t.Skip("test requires dev env - skipping it in short mode.")
	}

	s.cluster.Discover()
	require.InDelta(t, util.Timestamp(), s.cluster.LastDiscovered(), 1)

	elector := quorum.NewLagQuorum()
	logger := zerolog.New(zerolog.NewConsoleWriter())

	s.failover = NewDefaultFailover(s.cluster, FailoverConfig{
		Elector:                     elector,
		ReplicaSetRecoveryBlockTime: 2 * time.Second,
		InstanceRecoveryBlockTime:   2 * time.Second,
	}, logger)
	fv := s.failover.(*failover)

	stream := NewAnalysisStream()
	fv.Serve(stream)

	set, err := s.cluster.ReplicaSet("7432f072-c00b-4498-b1a6-6d9547a8a150")
	require.Nil(t, err)

	invalidUUID := "bd1095d1-1e73-4ceb-8e2f-6ebdc7838cb1"

	for i := range set.Instances {
		inst := &set.Instances[i]
		if inst.UUID == vshard.InstanceUUID(invalidUUID) {
			inst.VShardFingerprint = 100
			break
		}
	}

	analysis := &ReplicationAnalysis{
		Set:                         set,
		CountReplicas:               1,
		CountWorkingReplicas:        1,
		CountReplicatingReplicas:    1,
		CountInconsistentVShardConf: 1,
		State:                       InconsistentVShardConfiguration,
	}
	stream <- analysis

	require.Eventually(t, func() bool {
		return fv.hasBlockedRecovery(invalidUUID)
	}, 5*time.Second, 100*time.Millisecond)
	require.Len(t, fv.recoveries, 1)
	recv, ok := fv.recoveries[0].(*InstanceRecovery)
	require.True(t, ok)

	assert.True(t, recv.IsSuccessful)
	assert.Equal(t, string(analysis.State), recv.Reason())
	assert.Equal(t, invalidUUID, recv.LockKey())
	assert.False(t, recv.Expired())
}

func TestFailover(t *testing.T) {
	suite.Run(t, new(failoverTestSuite))
}
