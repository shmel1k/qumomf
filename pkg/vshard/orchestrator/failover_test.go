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

func (s *failoverTestSuite) Test_promoteFailover_promoteFollowerToMaster() {
	t := s.T()

	if testing.Short() {
		t.Skip("test requires dev env - skipping it in short mode.")
	}

	s.cluster.Discover()
	require.InDelta(t, util.Timestamp(), s.cluster.LastDiscovered(), 1)

	elector := quorum.NewLagQuorum()

	s.failover = NewPromoteFailover(s.cluster, FailoverConfig{
		Logger: zerolog.New(zerolog.NewConsoleWriter()),
		//Logger:                      zerolog.Nop(),
		Elector:                     elector,
		ReplicaSetRecoveryBlockTime: 2 * time.Second,
	})
	fv := s.failover.(*promoteFailover)

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
		return fv.hasBlockedRecovery(set.UUID)
	}, 5*time.Second, 100*time.Millisecond)
	require.Len(t, fv.blockers, 1)
	recv := fv.blockers[0].Recovery

	require.True(t, recv.IsSuccessful)
	assert.InDelta(t, util.Timestamp(), recv.StartTimestamp, 5)
	assert.InDelta(t, util.Timestamp(), recv.EndTimestamp, 2)
	assert.Equal(t, string(analysis.State), recv.Type)
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

	require.Len(t, fv.blockers, 1)
	assert.Same(t, recv, fv.blockers[0].Recovery)

	// Recreate the initial cluster.
	fv.cleanup(true)
	require.False(t, fv.hasBlockedRecovery(set.UUID))

	stream <- analysis

	require.Eventually(t, func() bool {
		return fv.hasBlockedRecovery(set.UUID)
	}, 5*time.Second, 100*time.Millisecond)
	require.Len(t, fv.blockers, 1)
	assert.True(t, recv != fv.blockers[0].Recovery)

	recv = fv.blockers[0].Recovery
	assert.True(t, recv.IsSuccessful)
	assert.Equal(t, set.MasterUUID, recv.SuccessorUUID)
}

func TestFailover(t *testing.T) {
	suite.Run(t, new(failoverTestSuite))
}
