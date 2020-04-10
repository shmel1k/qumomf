package config

import (
	"io/ioutil"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

const (
	defaultLogLevel               = "debug"
	defaultReadOnly               = true
	defaultUser                   = "guest"
	defaultPassword               = "guest"
	defaultConnectTimeout         = 1 * time.Second
	defaultRequestTimeout         = 1 * time.Second
	defaultClusterDiscoveryTime   = 5 * time.Second
	defaultClusterRecoveryTime    = 1 * time.Second
	defaultShardRecoveryBlockTime = 30 * time.Minute
)

type Config struct {
	// Qumomf is a set of global options determines qumomf's behavior.
	Qumomf struct {
		Port                   string        `yaml:"port"`
		LogLevel               string        `yaml:"log_level"`
		ReadOnly               bool          `yaml:"readonly"`
		ClusterDiscoveryTime   time.Duration `yaml:"cluster_discovery_time"`
		ClusterRecoveryTime    time.Duration `yaml:"cluster_recovery_time"`
		ShardRecoveryBlockTime time.Duration `yaml:"shard_recovery_block_time"`
	} `yaml:"qumomf"`

	// Connection contains the default connection options for each instance in clusters.
	// This options might be overridden by cluster-level options.
	Connection *ConnectConfig           `yaml:"connection,omitempty"`
	Clusters   map[string]ClusterConfig `yaml:"clusters"`
}

type ConnectConfig struct {
	User           *string        `yaml:"user"`
	Password       *string        `yaml:"password"`
	ConnectTimeout *time.Duration `yaml:"connect_timeout"`
	RequestTimeout *time.Duration `yaml:"request_timeout"`
}

type ClusterConfig struct {
	// Connection contains connection options which qumomf should
	// use to connect to routers and instances in the cluster.
	Connection *ConnectConfig `yaml:"connection,omitempty"`

	// ReadOnly indicates whether qumomf can run a failover
	// or should just observe the cluster topology.
	ReadOnly *bool `yaml:"readonly,omitempty"`

	// OverrideURIRules contains list of URI used in tarantool replication and
	// their mappings which will be used in connection pool by qumomf.
	//
	// Use it if qumomf should not connect to the instances by URI
	// obtained from the replication configuration during the auto discovery.
	OverrideURIRules map[string]string `yaml:"override_uri_rules,omitempty"`

	// Routers contains list of all cluster routers.
	//
	// All cluster nodes must share a common topology.
	// An administrator must ensure that the configurations are identical.
	// The administrator must provide list of all routers so qumomf will be able
	// to update their configuration when failover is running.
	// Otherwise failover might break topology.
	Routers []RouterConfig `yaml:"routers"`
}

type RouterConfig struct {
	Name string `yaml:"name"`
	Addr string `yaml:"addr"`
	UUID string `yaml:"uuid"`
}

func Setup(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var cfg Config
	cfg.withDefaults()
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	cfg.overrideEmptyByGlobalConfigs()

	return &cfg, nil
}

func (c *Config) withDefaults() {
	if c == nil {
		return
	}

	base := &c.Qumomf
	base.ReadOnly = defaultReadOnly
	base.LogLevel = defaultLogLevel
	base.ClusterDiscoveryTime = defaultClusterDiscoveryTime
	base.ClusterRecoveryTime = defaultClusterRecoveryTime
	base.ShardRecoveryBlockTime = defaultShardRecoveryBlockTime

	connection := &ConnectConfig{}
	connection.User = newString(defaultUser)
	connection.Password = newString(defaultPassword)
	connection.ConnectTimeout = newDuration(defaultConnectTimeout)
	connection.RequestTimeout = newDuration(defaultRequestTimeout)
	c.Connection = connection
}

func (c *Config) overrideEmptyByGlobalConfigs() {
	for clusterUUID, clusterCfg := range c.Clusters {
		if clusterCfg.ReadOnly == nil {
			clusterCfg.ReadOnly = newBool(c.Qumomf.ReadOnly)
		}

		if clusterCfg.Connection == nil {
			clusterCfg.Connection = c.Connection
		} else {
			opts := clusterCfg.Connection
			if opts.ConnectTimeout == nil {
				opts.ConnectTimeout = c.Connection.ConnectTimeout
			}
			if opts.RequestTimeout == nil {
				opts.RequestTimeout = c.Connection.RequestTimeout
			}
			if opts.User == nil {
				opts.User = c.Connection.User
			}
			if opts.Password == nil {
				opts.Password = c.Connection.Password
			}
		}

		c.Clusters[clusterUUID] = clusterCfg
	}
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
