package config

import (
	"io/ioutil"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

const (
	defaultLogLevel                  = "debug"
	defaultSysLogEnabled             = false
	defaultFileLoggingEnabled        = false
	defaultLogFilename               = "/var/log/qumomf.log"
	defaultLogFileMaxSize            = 256
	defaultLogFileMaxBackups         = 3
	defaultLogFileMaxAge             = 5
	defaultReadOnly                  = true
	defaultUser                      = "guest"
	defaultPassword                  = "guest"
	defaultConnectTimeout            = 500 * time.Millisecond
	defaultRequestTimeout            = 1 * time.Second
	defaultClusterDiscoveryTime      = 5 * time.Second
	defaultClusterRecoveryTime       = 1 * time.Second
	defaultShardRecoveryBlockTime    = 30 * time.Minute
	defaultInstanceRecoveryBlockTime = 10 * time.Minute
	defaultElectorType               = "smart"
	defaultShellCommand              = "bash"
	defaultHookTimeout               = 5 * time.Second
	defaultAsyncHookTimeout          = 10 * time.Minute
	defaultMaxFollowerLSNLag         = 1000
	defaultMaxFollowerIdle           = 5 * time.Minute
)

type Config struct {
	// Qumomf is a set of global options determines qumomf's behavior.
	Qumomf struct {
		Port                      string        `yaml:"port"`
		Logging                   Logging       `yaml:"logging"`
		ReadOnly                  bool          `yaml:"readonly"`
		ClusterDiscoveryTime      time.Duration `yaml:"cluster_discovery_time"`
		ClusterRecoveryTime       time.Duration `yaml:"cluster_recovery_time"`
		ShardRecoveryBlockTime    time.Duration `yaml:"shard_recovery_block_time"`
		InstanceRecoveryBlockTime time.Duration `yaml:"instance_recovery_block_time"`
		ElectionMode              string        `yaml:"elector"`
		ReasonableFollowerLSNLag  int64         `yaml:"reasonable_follower_lsn_lag"`
		ReasonableFollowerIdle    time.Duration `yaml:"reasonable_follower_idle"`
		Hooks                     struct {
			Shell                    string        `yaml:"shell"`
			PreFailover              []string      `yaml:"pre_failover"`
			PostSuccessfulFailover   []string      `yaml:"post_successful_failover"`
			PostUnsuccessfulFailover []string      `yaml:"post_unsuccessful_failover"`
			Timeout                  time.Duration `yaml:"timeout"`
			TimeoutAsync             time.Duration `yaml:"timeout_async"`
		} `yaml:"hooks"`
	} `yaml:"qumomf"`

	// Connection contains the default connection options for each instance in clusters.
	// This options might be overridden by cluster-level options.
	Connection *ConnectConfig           `yaml:"connection,omitempty"`
	Clusters   map[string]ClusterConfig `yaml:"clusters"`
}

type Logging struct {
	Level              string `yaml:"level"`
	SysLogEnabled      bool   `yaml:"syslog_enabled"`
	FileLoggingEnabled bool   `yaml:"file_enabled"`
	Filename           string `yaml:"file_name"`
	MaxSize            int    `yaml:"file_max_size"`    // megabytes
	MaxBackups         int    `yaml:"file_max_backups"` // files
	MaxAge             int    `yaml:"file_max_age"`     // days
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

	// ElectionMode is a master election mode of the given cluster.
	ElectionMode *string `yaml:"elector"`

	// OverrideURIRules contains list of URI used in tarantool replication and
	// their mappings which will be used in connection pool by qumomf.
	//
	// Use it if qumomf should not connect to the instances by URI
	// obtained from the replication configuration during the auto discovery.
	OverrideURIRules map[string]string `yaml:"override_uri_rules,omitempty"`

	// Priorities contains list of instances UUID and their priorities.
	Priorities map[string]int `yaml:"priorities,omitempty"`

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

	err = validate(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) withDefaults() {
	if c == nil {
		return
	}

	base := &c.Qumomf
	base.ReadOnly = defaultReadOnly

	base.Logging.Level = defaultLogLevel
	base.Logging.SysLogEnabled = defaultSysLogEnabled
	base.Logging.FileLoggingEnabled = defaultFileLoggingEnabled
	base.Logging.Filename = defaultLogFilename
	base.Logging.MaxSize = defaultLogFileMaxSize
	base.Logging.MaxBackups = defaultLogFileMaxBackups
	base.Logging.MaxAge = defaultLogFileMaxAge

	base.ClusterDiscoveryTime = defaultClusterDiscoveryTime
	base.ClusterRecoveryTime = defaultClusterRecoveryTime
	base.ShardRecoveryBlockTime = defaultShardRecoveryBlockTime
	base.InstanceRecoveryBlockTime = defaultInstanceRecoveryBlockTime
	base.ElectionMode = defaultElectorType
	base.ReasonableFollowerLSNLag = defaultMaxFollowerLSNLag
	base.ReasonableFollowerIdle = defaultMaxFollowerIdle
	base.Hooks.Shell = defaultShellCommand
	base.Hooks.Timeout = defaultHookTimeout
	base.Hooks.TimeoutAsync = defaultAsyncHookTimeout

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

		if clusterCfg.ElectionMode == nil {
			clusterCfg.ElectionMode = newString(c.Qumomf.ElectionMode)
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

func validate(c *Config) error {
	err := validateElector(&c.Qumomf.ElectionMode)
	if err != nil {
		return err
	}

	for _, clusterCfg := range c.Clusters {
		err = validateElector(clusterCfg.ElectionMode)
		if err != nil {
			return err
		}
	}

	return nil
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
