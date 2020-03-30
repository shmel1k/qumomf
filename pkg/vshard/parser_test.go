// +build integration

package vshard

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRouterInfo(t *testing.T) {
	conn := setupConnection("127.0.0.1:9301", ConnOptions{
		User:           "qumomf",
		Password:       "qumomf",
		UUID:           "router_1_uuid",
		ConnectTimeout: 1 * time.Second,
		QueryTimeout:   1 * time.Second,
	})

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp := conn.Exec(ctx, vshardRouterInfoQuery)
	if resp.Error != nil {
		require.Nil(t, resp.Error, resp.String())
	}

	info, err := ParseRouterInfo(resp.Data)
	require.Nil(t, err)

	assert.Equal(t, int64(0), info.Status)
	assert.Empty(t, info.Alerts)

	b := RouterBucket{
		AvailableRO: 0,
		AvailableRW: 10000,
		Unknown:     0,
		Unreachable: 0,
	}
	assert.Equal(t, b, info.Bucket)

	rs := RouterReplicaSetParameters{
		"7432f072-c00b-4498-b1a6-6d9547a8a150": RouterInstanceParameters{
			UUID:           "294e7310-13f0-4690-b136-169599e87ba0",
			Status:         InstanceAvailable,
			URI:            "qumomf@qumomf_1_m.ddk:3301",
			NetworkTimeout: 0.5,
		},
		"5065fb5f-5f40-498e-af79-43887ba3d1ec": RouterInstanceParameters{
			UUID:           "f3ef657e-eb9a-4730-b420-7ea78d52797d",
			Status:         InstanceAvailable,
			URI:            "qumomf@qumomf_2_m.ddk:3301",
			NetworkTimeout: 0.5,
		},
	}
	assert.Equal(t, rs, info.ReplicaSets)
}

func TestParseReplication(t *testing.T) {
	conn := setupConnection("127.0.0.1:9303", ConnOptions{
		User:           "qumomf",
		Password:       "qumomf",
		UUID:           "294e7310-13f0-4690-b136-169599e87ba0",
		ConnectTimeout: 1 * time.Second,
		QueryTimeout:   1 * time.Second,
	})

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp := conn.Exec(ctx, vshardBoxInfoQuery)
	if resp.Error != nil {
		require.Nil(t, resp.Error, resp.String())
	}

	data, err := ParseReplication(resp.Data)
	require.Nil(t, err)

	assert.Len(t, data, 2)

	master := data[0]
	assert.Equal(t, InstanceUUID("294e7310-13f0-4690-b136-169599e87ba0"), master.UUID)
	assert.Equal(t, "", master.URI) // No upstream data for master, URI must be set manually
	assert.Equal(t, int64(5045), master.LSN)

	replica := data[1]
	assert.Equal(t, InstanceUUID("cd1095d1-1e73-4ceb-8e2f-6ebdc7838cb1"), replica.UUID)
	assert.Equal(t, "qumomf@qumomf_1_s.ddk:3301", replica.URI)
	assert.Equal(t, int64(0), replica.LSN)
	require.NotNil(t, replica.Upstream)
	assert.Equal(t, UpstreamFollow, replica.Upstream.Status)
}

func TestParseStorageInfo(t *testing.T) {
	conn := setupConnection("127.0.0.1:9303", ConnOptions{
		User:           "qumomf",
		Password:       "qumomf",
		UUID:           "294e7310-13f0-4690-b136-169599e87ba0",
		ConnectTimeout: 1 * time.Second,
		QueryTimeout:   1 * time.Second,
	})

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp := conn.Exec(ctx, vshardStorageInfoQuery)
	if resp.Error != nil {
		require.Nil(t, resp.Error, resp.String())
	}

	data, err := ParseStorageInfo(resp.Data)
	require.Nil(t, err)

	assert.Equal(t, StatusMaster, data.ReplicationStatus)
	assert.Empty(t, data.Alerts)

	b := InstanceBucket{
		Active:    5000,
		Garbage:   0,
		Pinned:    0,
		Receiving: 0,
		Sending:   0,
		Total:     5000,
	}
	assert.Equal(t, b, data.Bucket)
}
