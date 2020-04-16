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

	assert.Equal(t, ":8080", cfg.Qumomf.Port)
	assert.Equal(t, "debug", cfg.Qumomf.LogLevel)
	assert.True(t, cfg.Qumomf.ReadOnly)
	assert.Equal(t, 60*time.Second, cfg.Qumomf.ClusterDiscoveryTime)
	assert.Equal(t, 5*time.Second, cfg.Qumomf.ClusterRecoveryTime)

	assert.Equal(t, 500*time.Millisecond, *cfg.Connection.ConnectTimeout)
	assert.Equal(t, 1*time.Second, *cfg.Connection.RequestTimeout)

	connOpts := cfg.Connection
	require.NotNil(t, connOpts)
	assert.Equal(t, "qumomf", *connOpts.User)
	assert.Equal(t, "qumomf", *connOpts.Password)
	assert.Equal(t, 500*time.Millisecond, *connOpts.ConnectTimeout)
	assert.Equal(t, 1*time.Second, *connOpts.RequestTimeout)

	expected := map[string]ClusterConfig{
		"qumomf_sandbox_1": {
			Connection: &ConnectConfig{
				User:           newString("qumomf"),
				Password:       newString("qumomf"),
				ConnectTimeout: newDuration(500 * time.Millisecond),
				RequestTimeout: newDuration(1 * time.Second),
			},
			ReadOnly: newBool(false),
			OverrideURIRules: map[string]string{
				"qumomf_1_m.ddk:3301": "127.0.0.1:9303",
			},
			Routers: []RouterConfig{
				{
					Name: "sandbox1-router1",
					Addr: "127.0.0.1:9301",
					UUID: "294e7310-13f0-4690-b136-169599e87ba0",
				},
				{
					Name: "sandbox1-router2",
					Addr: "127.0.0.1:9302",
					UUID: "f3ef657e-eb9a-4730-b420-7ea78d52797d",
				},
			},
		},
		"qumomf_sandbox_2": {
			Connection: &ConnectConfig{
				User:           newString("tnt"),
				Password:       newString("tnt"),
				ConnectTimeout: newDuration(10 * time.Second),
				RequestTimeout: newDuration(10 * time.Second),
			},
			ReadOnly: newBool(true),
			Routers: []RouterConfig{
				{
					Name: "sandbox2-router1",
					Addr: "127.0.0.1:7301",
					UUID: "38dbe90b-9bca-4766-a98c-f02e56ddf986",
				},
			},
		},
	}

	assert.Equal(t, expected, cfg.Clusters)
}
