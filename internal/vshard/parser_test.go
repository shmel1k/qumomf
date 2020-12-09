package vshard

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRouterInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("test requires dev env - skipping it in short mode.")
	}

	conn := setupConnection("127.0.0.1:9301", ConnOptions{
		User:           "qumomf",
		Password:       "qumomf",
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
		AvailableRW: 120,
		Unknown:     0,
		Unreachable: 0,
	}
	assert.Equal(t, b, info.Bucket)

	expected := RouterReplicaSetParameters{
		"7432f072-c00b-4498-b1a6-6d9547a8a150": RouterInstanceParameters{
			UUID:           "a94e7310-13f0-4690-b136-169599e87ba0",
			Status:         InstanceAvailable,
			URI:            "qumomf_1_m.ddk:3301",
			NetworkTimeout: 0.5,
		},
		"5065fb5f-5f40-498e-af79-43887ba3d1ec": RouterInstanceParameters{
			UUID:           "a3ef657e-eb9a-4730-b420-7ea78d52797d",
			Status:         InstanceAvailable,
			URI:            "qumomf_2_m.ddk:3301",
			NetworkTimeout: 0.5,
		},
	}

	require.Len(t, info.ReplicaSets, len(expected))
	for uuid, set := range info.ReplicaSets {
		expSet, ok := expected[uuid]
		require.True(t, ok)

		assert.Equal(t, expSet.UUID, set.UUID)
		assert.Equal(t, expSet.Status, set.Status)
		assert.Equal(t, expSet.URI, set.URI)
		assert.InDelta(t, expSet.NetworkTimeout, set.NetworkTimeout, 1.0)
	}
}

func TestParseReplication(t *testing.T) {
	if testing.Short() {
		t.Skip("test requires dev env - skipping it in short mode.")
	}

	conn := setupConnection("127.0.0.1:9303", ConnOptions{
		User:           "qumomf",
		Password:       "qumomf",
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
	assert.Equal(t, uint64(1), master.ID)
	assert.Equal(t, InstanceUUID("a94e7310-13f0-4690-b136-169599e87ba0"), master.UUID)
	assert.Equal(t, "", master.URI) // No upstream data for master, URI must be set manually
	assert.Equal(t, int64(105), master.LSN)
	assert.Equal(t, int64(0), master.LSNBehindMaster)
	assert.Nil(t, master.Upstream)
	assert.Nil(t, master.Downstream)

	replica := data[1]
	assert.Equal(t, uint64(2), replica.ID)
	assert.Equal(t, InstanceUUID("bd1095d1-1e73-4ceb-8e2f-6ebdc7838cb1"), replica.UUID)
	assert.Equal(t, "qumomf_1_s.ddk:3301", replica.URI)
	assert.Equal(t, int64(0), replica.LSN)
	assert.Equal(t, int64(0), replica.LSNBehindMaster)
	require.NotNil(t, replica.Upstream)
	assert.Equal(t, UpstreamFollow, replica.Upstream.Status)
	require.NotNil(t, replica.Downstream)
	assert.Equal(t, DownstreamFollow, replica.Downstream.Status)
}

func TestParseInstanceInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("test requires dev env - skipping it in short mode.")
	}

	conn := setupConnection("127.0.0.1:9304", ConnOptions{
		User:           "qumomf",
		Password:       "qumomf",
		ConnectTimeout: 1 * time.Second,
		QueryTimeout:   1 * time.Second,
	})

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp := conn.Exec(ctx, vshardInstanceInfoQuery)
	if resp.Error != nil {
		require.Nil(t, resp.Error, resp.String())
	}

	data, err := ParseInstanceInfo(resp.Data)
	require.Nil(t, err)

	assert.True(t, data.Readonly)
	assert.Equal(t, uint64(251215738), data.VShardFingerprint)

	storage := &data.StorageInfo
	assert.Equal(t, HealthCodeGreen, storage.Status)

	replication := &storage.Replication
	assert.Equal(t, StatusFollow, replication.Status)

	assert.Empty(t, storage.Alerts)

	b := InstanceBucket{
		Active:    60,
		Garbage:   0,
		Pinned:    0,
		Receiving: 0,
		Sending:   0,
		Total:     60,
	}
	assert.Equal(t, b, storage.Bucket)
}

func TestParseReplication_TableTests(t *testing.T) {
	tests := []struct {
		name string
		data [][]interface{}
		want []Instance
	}{
		{
			name: "regular_case",
			data: [][]interface{}{
				{
					map[string]interface{}{
						"id":                int64(1),
						"uuid":              "uuid",
						"lsn":               int64(1),
						"lsn_behind_master": int64(1),
						"downstream": map[string]interface{}{
							"idle":   int64(1),
							"status": "follow",
						},
						"upstream": map[string]interface{}{
							"idle":   int64(1),
							"lag":    int64(1),
							"peer":   "test@test",
							"status": "follow",
						},
					},
				},
			},
			want: []Instance{
				{
					URI:             "test",
					ID:              1,
					UUID:            "uuid",
					LSN:             1,
					LSNBehindMaster: 1,
					Downstream:      &Downstream{Status: DownstreamFollow},
					Upstream: &Upstream{
						Status: UpstreamFollow,
						Idle:   1,
						Peer:   "test@test",
						Lag:    1,
					},
				},
			},
		},
		{
			name: "response_array_with_gaps_should_ignore_it",
			data: [][]interface{}{
				{
					nil,
					map[string]interface{}{
						"id":                int64(1),
						"uuid":              "uuid",
						"lsn":               int64(1),
						"lsn_behind_master": int64(1),
					},
				},
			},
			want: []Instance{
				{
					ID:              1,
					UUID:            "uuid",
					LSN:             1,
					LSNBehindMaster: 1,
				},
			},
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseReplication(tc.data)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
