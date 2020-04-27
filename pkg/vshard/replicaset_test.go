package vshard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplicaSet_Followers(t *testing.T) {
	type fields struct {
		UUID       ReplicaSetUUID
		MasterUUID InstanceUUID
		Instances  []Instance
	}

	tests := []struct {
		name   string
		fields fields
		want   []InstanceUUID
	}{
		{
			name: "NoFollowers",
			fields: fields{
				UUID:       "uuid_1",
				MasterUUID: "master_uuid_1",
				Instances:  []Instance{},
			},
			want: []InstanceUUID{},
		},
		{
			name: "MultipleFollowers",
			fields: fields{
				UUID:       "uuid_1",
				MasterUUID: "master_uuid_1",
				Instances: []Instance{
					{
						UUID: "master_uuid_1",
					},
					{
						UUID: "replica_uuid_1",
					},
					{
						UUID: "replica_uuid_2",
					},
				},
			},
			want: []InstanceUUID{"replica_uuid_1", "replica_uuid_2"},
		},
	}

	for _, tv := range tests {
		tt := tv
		t.Run(tt.name, func(t *testing.T) {
			set := ReplicaSet{
				UUID:       tt.fields.UUID,
				MasterUUID: tt.fields.MasterUUID,
				Instances:  tt.fields.Instances,
			}

			followers := set.Followers()
			got := make([]InstanceUUID, 0, len(followers))
			for _, f := range followers {
				got = append(got, f.UUID)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReplicaSet_AliveFollowers(t *testing.T) {
	type fields struct {
		UUID       ReplicaSetUUID
		MasterUUID InstanceUUID
		Instances  []Instance
	}

	tests := []struct {
		name   string
		fields fields
		want   []InstanceUUID
	}{
		{
			name: "NoFollowers",
			fields: fields{
				UUID:       "uuid_1",
				MasterUUID: "master_uuid_1",
				Instances:  []Instance{},
			},
			want: []InstanceUUID{},
		},
		{
			name: "MultipleFollowers",
			fields: fields{
				UUID:       "uuid_1",
				MasterUUID: "master_uuid_1",
				Instances: []Instance{
					{
						UUID: "master_uuid_1",
						Upstream: &Upstream{
							Status: UpstreamRunning,
						},
					},
					{
						UUID:           "replica_uuid_1",
						LastCheckValid: true,
						Upstream: &Upstream{
							Status: UpstreamFollow,
						},
						Downstream: &Downstream{
							Status: DownstreamFollow,
						},
					},
					{
						UUID:           "replica_uuid_2",
						LastCheckValid: true,
						Upstream: &Upstream{
							Status: UpstreamFollow,
						},
						Downstream: &Downstream{
							Status: DownstreamFollow,
						},
					},
					{
						UUID:           "replica_uuid_3",
						LastCheckValid: true,
						Upstream: &Upstream{
							Status: UpstreamStopped,
						},
					},
					{
						UUID:           "replica_uuid_4",
						LastCheckValid: false,
						Upstream: &Upstream{
							Status: UpstreamFollow,
						},
					},
				},
			},
			want: []InstanceUUID{"replica_uuid_1", "replica_uuid_2"},
		},
	}

	for _, tv := range tests {
		tt := tv
		t.Run(tt.name, func(t *testing.T) {
			set := &ReplicaSet{
				UUID:       tt.fields.UUID,
				MasterUUID: tt.fields.MasterUUID,
				Instances:  tt.fields.Instances,
			}

			followers := set.AliveFollowers()
			got := make([]InstanceUUID, 0, len(followers))
			for _, f := range followers {
				got = append(got, f.UUID)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
