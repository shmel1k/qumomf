package quorum

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shmel1k/qumomf/internal/vshard"
)

func Test_filter(t *testing.T) {
	tests := []struct {
		name      string
		opts      Options
		instances []vshard.Instance
		want      []vshard.InstanceUUID
	}{
		{
			name: "ExcludeByPriority",
			opts: Options{},
			instances: []vshard.Instance{
				{
					UUID:     "1",
					Priority: -1,
				},
				{
					UUID:     "2",
					Priority: 0,
				},
				{
					UUID:     "3",
					Priority: 1,
				},
			},
			want: []vshard.InstanceUUID{
				"2", "3",
			},
		},
		{
			name: "ExcludeByLSN",
			opts: Options{
				ReasonableFollowerLSNLag: 100,
			},
			instances: []vshard.Instance{
				{
					UUID:            "1",
					LSNBehindMaster: 1000,
				},
				{
					UUID:            "2",
					LSNBehindMaster: 100,
				},
				{
					UUID:            "3",
					LSNBehindMaster: 0,
				},
			},
			want: []vshard.InstanceUUID{
				"2", "3",
			},
		},
		{
			name: "ExcludeByIdle",
			opts: Options{
				ReasonableFollowerIdle: 5.5,
			},
			instances: []vshard.Instance{
				{
					UUID: "1",
					Upstream: &vshard.Upstream{
						Status: vshard.UpstreamFollow,
						Idle:   7.2,
					},
				},
				{
					UUID: "2",
					Upstream: &vshard.Upstream{
						Status: vshard.UpstreamFollow,
						Idle:   5.1,
					},
				},
				{
					UUID: "3",
					Upstream: &vshard.Upstream{
						Status: vshard.UpstreamFollow,
						Idle:   0.86981821060181,
					},
				},
			},
			want: []vshard.InstanceUUID{
				"2", "3",
			},
		},
		{
			name: "ExcludeAll",
			opts: Options{
				ReasonableFollowerLSNLag: 100,
				ReasonableFollowerIdle:   5.5,
			},
			instances: []vshard.Instance{
				{
					UUID:            "1",
					Priority:        0,
					LSNBehindMaster: 10,
					Upstream: &vshard.Upstream{
						Status: vshard.UpstreamFollow,
						Idle:   7.2,
					},
				},
				{
					UUID:            "2",
					Priority:        -1,
					LSNBehindMaster: 0,
					Upstream: &vshard.Upstream{
						Status: vshard.UpstreamFollow,
						Idle:   0.2,
					},
				},
				{
					UUID:            "3",
					Priority:        100,
					LSNBehindMaster: 1000,
					Upstream: &vshard.Upstream{
						Status: vshard.UpstreamFollow,
						Idle:   0.1,
					},
				},
			},
			want: []vshard.InstanceUUID{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := filter(tt.instances, tt.opts)
			uuids := make([]vshard.InstanceUUID, len(got))
			for i, inst := range got {
				uuids[i] = inst.UUID
			}
			assert.Equal(t, tt.want, uuids)
		})
	}
}
