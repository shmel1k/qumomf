package vshard

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shmel1k/qumomf/internal/config"
	"github.com/shmel1k/qumomf/pkg/util"
)

type tExpSet struct {
	setUUID    ReplicaSetUUID
	masterUUID InstanceUUID
	instances  []tExpInst
}

type tExpInst struct {
	uuid              InstanceUUID
	uri               string
	readonly          bool
	hasUpstream       bool
	upstreamStatus    UpstreamStatus
	upstreamPeer      string
	replicationStatus ReplicationStatus
}

func TestCluster_Discover(t *testing.T) {
	if testing.Short() {
		t.Skip("test requires dev env - skipping it in short mode.")
	}

	c := mockCluster()
	c.Discover()

	assert.InDelta(t, util.Timestamp(), c.LastDiscovered(), 1000)

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
					readonly:          false,
					hasUpstream:       false,
					replicationStatus: StatusMaster,
				},
				{
					uuid:              "cd1095d1-1e73-4ceb-8e2f-6ebdc7838cb1",
					uri:               "qumomf@qumomf_1_s.ddk:3301",
					readonly:          true,
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
					readonly:          false,
					hasUpstream:       false,
					replicationStatus: StatusMaster,
				},
				{
					uuid:              "7d64dd00-161e-4c99-8b3c-d3c4635e18d2",
					uri:               "qumomf@qumomf_2_s.ddk:3301",
					readonly:          true,
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
			assert.Equal(t, expInst.readonly, inst.Readonly)
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

func TestCluster_Instance(t *testing.T) {
	sets := []ReplicaSet{
		{
			UUID:       "set_1",
			MasterUUID: "replica_1",
			Instances: []Instance{
				{
					UUID: "set_1_replica_1",
				},
				{
					UUID: "set_1_replica_2",
				},
				{
					UUID: "set_1_replica_3",
				},
			},
		},
		{
			UUID:       "set_2",
			MasterUUID: "replica_2",
			Instances: []Instance{
				{
					UUID: "set_2_replica_1",
				},
				{
					UUID: "set_2_replica_2",
				},
			},
		},
	}

	c := mockCluster()
	c.snapshot = Snapshot{
		Created:     util.Timestamp(),
		Routers:     c.Routers(),
		ReplicaSets: sets,
	}

	tests := []struct {
		name    string
		uuid    InstanceUUID
		wantErr bool
	}{
		{
			name:    "KnownUUID_ShouldReturnInstance",
			uuid:    "set_2_replica_1",
			wantErr: false,
		},
		{
			name:    "UnknownUUID_ShouldReturnErr",
			uuid:    "set_2_replica_1000",
			wantErr: true,
		},
	}

	for _, tv := range tests {
		tt := tv
		t.Run(tt.name, func(t *testing.T) {
			inst, err := c.Instance(tt.uuid)
			if tt.wantErr {
				require.NotNil(t, err)
				assert.Equal(t, ErrInstanceNotFound, err)
			} else {
				require.Nil(t, err)
				assert.Equal(t, tt.uuid, inst.UUID)
			}
		})
	}
}

func mockCluster() *Cluster {
	return NewCluster("sandbox", config.ClusterConfig{
		Connection: &config.ConnectConfig{
			User:           util.NewString("qumomf"),
			Password:       util.NewString("qumomf"),
			ConnectTimeout: util.NewDuration(1 * time.Second),
			RequestTimeout: util.NewDuration(1 * time.Second),
		},
		ReadOnly: util.NewBool(true),
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
}
