package config

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetup_InvalidPath(t *testing.T) {
	cfg, err := Setup("invalid_path")
	assert.NotNil(t, err)
	assert.Nil(t, cfg)
}

func TestSetup_ValidPath(t *testing.T) {
	testConfigPath, err := filepath.Abs("testdata/qumomf-full.conf.yaml")
	require.Nil(t, err)

	cfg, err := Setup(testConfigPath)
	require.Nil(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, ":1488", cfg.Qumomf.Port)

	assert.Equal(t, 1*time.Second, cfg.Tarantool.RequestTimeout)
	assert.Equal(t, 1*time.Second, cfg.Tarantool.ConnectTimeout)

	topology := cfg.Tarantool.Topology
	assert.True(t, topology.ReadOnly)
	assert.Equal(t, "qumomf", topology.User)
	assert.Equal(t, "qumomf", topology.Password)

	expected := map[string]ClusterConfig{
		"qumomf_sandbox_1": {
			ReadOnly: newBool(false),
			Shards: map[string][]InstanceConfig{
				"7432f072-c00b-4498-b1a6-6d9547a8a150": {
					{
						Name:           "qumomf_1_m",
						Addr:           "127.0.0.1:9303",
						UUID:           "294e7310-13f0-4690-b136-169599e87ba0",
						User:           newString("qumomf"),
						Password:       newString("qumomf"),
						ConnectTimeout: newDuration(1 * time.Second),
						RequestTimeout: newDuration(1 * time.Second),
						Master:         true,
					},
				},
				"5065fb5f-5f40-498e-af79-43887ba3d1ec": {
					{
						Name:           "qumomf_2_m",
						Addr:           "127.0.0.1:9305",
						UUID:           "f3ef657e-eb9a-4730-b420-7ea78d52797d",
						User:           newString("qumomf"),
						Password:       newString("qumomf"),
						ConnectTimeout: newDuration(1 * time.Second),
						RequestTimeout: newDuration(1 * time.Second),
						Master:         true,
					},
					{
						Name:           "qumomf_2_s",
						Addr:           "127.0.0.1:9306",
						UUID:           "7d64dd00-161e-4c99-8b3c-d3c4635e18d2",
						User:           newString("qumomf"),
						Password:       newString("qumomf"),
						ConnectTimeout: newDuration(1 * time.Second),
						RequestTimeout: newDuration(1 * time.Second),
						Master:         false,
					},
				},
			},
			Routers: []InstanceConfig{
				{
					Name:           "router_1",
					Addr:           "127.0.0.1:9301",
					UUID:           "router_1_uuid",
					User:           newString("qumomf"),
					Password:       newString("qumomf"),
					ConnectTimeout: newDuration(1 * time.Second),
					RequestTimeout: newDuration(1 * time.Second),
					Master:         false,
				},
			},
		},
		"qumomf_sandbox_2": {
			ReadOnly: newBool(true),
			Routers: []InstanceConfig{
				{
					Name:           "router_2",
					Addr:           "127.0.0.1:7301",
					UUID:           "router_2_uuid",
					User:           newString("tnt"),
					Password:       newString("tnt"),
					ConnectTimeout: newDuration(10 * time.Second),
					RequestTimeout: newDuration(10 * time.Second),
					Master:         false,
				},
			},
		},
	}

	assert.Equal(t, expected, cfg.Clusters)
}

func newBool(v bool) *bool {
	return &v
}

func newDuration(v time.Duration) *time.Duration {
	return &v
}

func newString(v string) *string {
	return &v
}
