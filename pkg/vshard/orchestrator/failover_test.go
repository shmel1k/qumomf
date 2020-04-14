package orchestrator

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shmel1k/qumomf/internal/config"
	"github.com/shmel1k/qumomf/pkg/quorum"
	"github.com/shmel1k/qumomf/pkg/util"
	"github.com/shmel1k/qumomf/pkg/vshard"
)

func Test_swapMasterFailover_promoteFollowerToMaster(t *testing.T) {
	if testing.Short() {
		t.Skip("test requires dev env - skipping it in short mode.")
	}

	c := vshard.NewCluster("sandbox", config.ClusterConfig{
		Connection: &config.ConnectConfig{
			User:           util.NewString("qumomf"),
			Password:       util.NewString("qumomf"),
			ConnectTimeout: util.NewDuration(1 * time.Second),
			RequestTimeout: util.NewDuration(1 * time.Second),
		},
		ReadOnly: util.NewBool(false),
		OverrideURIRules: map[string]string{
			"qumomf_1_m.ddk:3301": "127.0.0.1:9303",
			"qumomf_1_s.ddk:3301": "127.0.0.1:9304",
			"qumomf_2_m.ddk:3301": "127.0.0.1:9305",
			"qumomf_2_s.ddk:3301": "127.0.0.1:9306",
		},
		Routers: []config.RouterConfig{
			{
				Name: "router_1",
				Addr: "127.0.0.1:9301",
				UUID: "router_uuid_1",
			},
		},
	})
	defer c.Shutdown()

	c.Discover()
	require.InDelta(t, util.Timestamp(), c.LastDiscovered(), 1)

	elector := quorum.NewLagQuorum()

	var fv *promoteFailover
	{
		failover := NewPromoteFailover(c, FailoverConfig{
			Logger:                      zerolog.Nop(),
			Elector:                     elector,
			ReplicaSetRecoveryBlockTime: 2 * time.Second,
		})
		fv = failover.(*promoteFailover)
	}

	stream := NewAnalysisStream()
	fv.Serve(stream)
	defer fv.Shutdown()

	set, err := c.ReplicaSet("7432f072-c00b-4498-b1a6-6d9547a8a150")
	require.Nil(t, err)

	analysis := &ReplicationAnalysis{
		Set:                      set,
		CountReplicas:            1,
		CountWorkingReplicas:     0,
		CountReplicatingReplicas: 0,
		State:                    DeadMaster,
	}
	stream <- analysis

	time.Sleep(200 * time.Millisecond)

	require.True(t, fv.hasBlockedRecovery(set.UUID))
	require.Len(t, fv.blockers, 1)
	recv := fv.blockers[0].Recovery

	assert.InDelta(t, util.Timestamp(), recv.StartTimestamp, 1)
	assert.InDelta(t, util.Timestamp(), recv.EndTimestamp, 1)
	assert.True(t, recv.IsSuccessful)
	assert.Equal(t, string(analysis.State), recv.Type)
	assert.Equal(t, set.MasterUUID, recv.FailedUUID)

	recvSet, err := c.ReplicaSet("7432f072-c00b-4498-b1a6-6d9547a8a150")
	require.Nil(t, err)

	assert.Equal(t, recv.SuccessorUUID, recvSet.MasterUUID)

	master, err := recvSet.Master()
	require.Nil(t, err)
	assert.False(t, master.Readonly)

	alive := recvSet.AliveFollowers()
	assert.Len(t, alive, 1)
	for _, f := range alive {
		assert.True(t, f.Readonly)
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

	require.Len(t, fv.blockers, 1)
	assert.True(t, recv != fv.blockers[0].Recovery)
}
