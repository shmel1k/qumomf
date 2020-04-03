package vshard

import (
	"sort"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shmel1k/qumomf/internal/config"
)

type tExpSet struct {
	setUUID    ReplicaSetUUID
	masterUUID InstanceUUID
	instances  []tExpInst
}

type tExpInst struct {
	uuid              InstanceUUID
	uri               string
	hasUpstream       bool
	upstreamStatus    UpstreamStatus
	upstreamPeer      string
	replicationStatus ReplicationStatus
}

func TestCluster_Discover(t *testing.T) {
	if testing.Short() {
		t.Skip("test requires dev env - skipping it in short mode.")
	}

	c := NewCluster("sandbox", config.ClusterConfig{
		Connection: &config.ConnectConfig{
			User:           newString("qumomf"),
			Password:       newString("qumomf"),
			ConnectTimeout: newDuration(1 * time.Second),
			RequestTimeout: newDuration(1 * time.Second),
		},
		ReadOnly: newBool(true),
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

	lvl := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.Disabled)
	c.Discover()
	zerolog.SetGlobalLevel(lvl)

	assert.InDelta(t, timestamp(), c.LastDiscovered(), 1000)

	routers := c.Routers()
	require.Len(t, routers, 1)
	r := routers[0]
	assert.Equal(t, RouterUUID("router_uuid_1"), r.UUID)
	assert.Equal(t, "127.0.0.1:9301", r.URI)

	sets := c.ReplicaSets()
	sort.SliceStable(sets, func(i, j int) bool { // predictable order
		return sets[j].UUID < sets[i].UUID
	})

	expected := []tExpSet{
		{
			setUUID:    "7432f072-c00b-4498-b1a6-6d9547a8a150",
			masterUUID: "294e7310-13f0-4690-b136-169599e87ba0",
			instances: []tExpInst{
				{
					uuid:              "294e7310-13f0-4690-b136-169599e87ba0",
					uri:               "qumomf@qumomf_1_m.ddk:3301",
					hasUpstream:       false,
					replicationStatus: StatusMaster,
				},
				{
					uuid:              "cd1095d1-1e73-4ceb-8e2f-6ebdc7838cb1",
					uri:               "qumomf@qumomf_1_s.ddk:3301",
					hasUpstream:       true,
					upstreamStatus:    UpstreamFollow,
					upstreamPeer:      "qumomf@qumomf_1_s.ddk:3301",
					replicationStatus: StatusFollow,
				},
			},
		},
		{
			setUUID:    "5065fb5f-5f40-498e-af79-43887ba3d1ec",
			masterUUID: "f3ef657e-eb9a-4730-b420-7ea78d52797d",
			instances: []tExpInst{
				{
					uuid:              "f3ef657e-eb9a-4730-b420-7ea78d52797d",
					uri:               "qumomf@qumomf_2_m.ddk:3301",
					hasUpstream:       false,
					replicationStatus: StatusMaster,
				},
				{
					uuid:              "7d64dd00-161e-4c99-8b3c-d3c4635e18d2",
					uri:               "qumomf@qumomf_2_s.ddk:3301",
					hasUpstream:       true,
					upstreamStatus:    UpstreamFollow,
					upstreamPeer:      "qumomf@qumomf_2_s.ddk:3301",
					replicationStatus: StatusFollow,
				},
			},
		},
	}

	require.Len(t, sets, len(expected))

	for i, set := range sets {
		exp := expected[i]

		assert.Equal(t, exp.setUUID, set.UUID)
		assert.Equal(t, exp.masterUUID, set.MasterUUID)

		require.Len(t, set.Instances, len(exp.instances))

		for j, inst := range set.Instances {
			expInst := exp.instances[j]

			assert.Equal(t, expInst.uuid, inst.UUID)
			assert.Equal(t, expInst.uri, inst.URI)
			assert.True(t, inst.LastCheckValid)

			upstream := inst.Upstream
			if expInst.hasUpstream {
				assert.NotNil(t, upstream)
				assert.Equal(t, expInst.upstreamStatus, upstream.Status)
				assert.Equal(t, expInst.upstreamPeer, inst.Upstream.Peer)
				assert.Empty(t, inst.Upstream.Message)
			} else {
				assert.Nil(t, upstream)
			}

			assert.Equal(t, expInst.replicationStatus, inst.StorageInfo.Replication.Status)
		}
	}
}
