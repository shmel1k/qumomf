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
	testConfigPath, err := filepath.Abs("testdata/qumomf-full.conf.yml")
	require.Nil(t, err)

	cfg, err := Setup(testConfigPath)
	require.Nil(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, ":8080", cfg.Qumomf.Port)
	assert.Equal(t, "debug", cfg.Qumomf.LogLevel)
	assert.True(t, cfg.Qumomf.ReadOnly)
	assert.Equal(t, 60*time.Second, cfg.Qumomf.ClusterDiscoveryTime)
	assert.Equal(t, 5*time.Second, cfg.Qumomf.ClusterRecoveryTime)
	assert.Equal(t, 30*time.Minute, cfg.Qumomf.ShardRecoveryBlockTime)
	assert.Equal(t, 10*time.Minute, cfg.Qumomf.InstanceRecoveryBlockTime)

	hooks := cfg.Qumomf.Hooks
	assert.Equal(t, "bash", hooks.Shell)
	assert.Equal(t, 5*time.Second, hooks.Timeout)
	assert.Equal(t, 10*time.Minute, hooks.TimeoutAsync)
	assert.Equal(t, []string{"echo 'Will recover from {failureType} on {failureCluster}' >> /tmp/qumomf_recovery.log"}, hooks.PreFailover)
	assert.Equal(t, []string{"echo 'Recovered from {failureType} on {failureCluster}. Set: {failureReplicaSetUUID}; Failed: {failedURI}; Successor: {successorURI}' >> /tmp/qumomf_recovery.log"}, hooks.PostSuccessfulFailover)
	assert.Equal(t, []string{"echo 'Failed to recover from {failureType} on {failureCluster}. Set: {failureReplicaSetUUID}; Failed: {failedURI}' >> /tmp/qumomf_recovery.log"}, hooks.PostUnsuccessfulFailover)

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
			ReadOnly:     newBool(false),
			ElectionMode: newString("smart"),
			OverrideURIRules: map[string]string{
				"qumomf_1_m.ddk:3301": "127.0.0.1:9303",
			},
			Routers: []RouterConfig{
				{
					Name: "sandbox1-router1",
					Addr: "127.0.0.1:9301",
					UUID: "a94e7310-13f0-4690-b136-169599e87ba0",
				},
				{
					Name: "sandbox1-router2",
					Addr: "127.0.0.1:9302",
					UUID: "a3ef657e-eb9a-4730-b420-7ea78d52797d",
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
			ReadOnly:     newBool(true),
			ElectionMode: newString("delay"),
			Priorities: map[string]int{
				"bd64dd00-161e-4c99-8b3c-d3c4635e18d2": 10,
				"cc4cfb9c-11d8-4810-84d2-66cfbebb0f6e": 5,
				"a3ef657e-eb9a-4730-b420-7ea78d52797d": -1,
			},
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

func TestSetup_InvalidElectorOption(t *testing.T) {
	testConfigPath, err := filepath.Abs("testdata/bad-elector.conf.yml")
	require.Nil(t, err)

	cfg, err := Setup(testConfigPath)
	require.NotNil(t, err)
	assert.Nil(t, cfg)
}
