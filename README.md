![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/shmel1k/qumomf?sort=semver&style=for-the-badge)
![GitHub Workflow Status](https://img.shields.io/github/workflow/status/shmel1k/qumomf/CI?style=for-the-badge)

# Qumomf

Qumomf is a Tarantool vshard high availability tool which supports discovery and recovery.

# Table of Contents

  * [Discovery](#discovery)
  * [Configuration](#configuration)
     * [How to add a new cluster](#how-to-add-a-new-cluster)
  * [Topology recovery](#topology-recovery)
     * [Idle](#idle)
     * [Smart](#smart)
  * [Recovery hooks](#recovery-hooks)
     * [Hooks arguments and environment](#hooks-arguments-and-environment)
  * [API](#api)
  * [Hacking](#hacking)

## Discovery

Qumomf actively crawls through your topologies and analyzes them. 
It reads basic vshard info such as replication status and configuration.

You should provide at least one router which will be an entrypoint to the discovery process.

## Configuration

For a sample qumomf configuration and its description see [example](config/qumomf.conf.yml).

### How to add a new cluster

Edit your configuration file and add a new cluster, e.g.:

```yaml
clusters:
  my_cluster:
    routers:
      - name: 'my_cluster_router_1'
        addr: 'localhost:3301'
```

You might override default connection settings for each cluster.

```yaml
clusters:
  my_cluster:
    connection:
      user: 'tnt'
      password: 'tnt'
      connect_timeout: 10s
      request_timeout: 10s

    routers:
      - name: 'my_cluster_router_1'
        addr: 'localhost:3301'
```

For a sample vshard configuration, 
see [qumomf example](/example) or [Tarantool documentation](https://www.tarantool.io/en/doc/1.10/reference/reference_rock/vshard/vshard_quick/#vshard-config-cluster-example).

Start qumomf, and it will discover all clusters defined in the configuration.

## Topology recovery

Just now qumomf supports only automated master recovery.
It is a configurable option and can be disabled completely or for a cluster via configuration.

Master election supports two modes: `idle` and `smart`.
Election mode might be configured for each cluster independently.

Both electors supports those options:

  - `reasonable_follower_lsn_lag` - on crash recovery, followers that are lagging 
     more than given LSN must not participate in the election.
  - `reasonable_follower_idle` - on crash recovery, followers that are lagging 
     more than given duration must not participate in the election.

Value of 0 disables this features.

### Idle

Naive and simple elector which finds alive replica last communicated to the failed master (received data or heartbeat signal).
Followers with the negative priority will be excluded from the master election.

### Smart

Elector tries to involve as many metrics as can:
  - vshard configuration consistency (prefer replica which has the same configuration as master), 
  - which upstream status did replica have before the crash,
  - how replica is far from the master comparing LSN to the master LSN,
  - last time when replica received data or heartbeat signal from the master,
  - user promotion rules based on the instance priorities.

You can define your own promotion rules which will influence on master election during a failover.
Each instance has a priority set via config. Negative priority excludes follower from the election process. 

## Recovery hooks

Hooks invoked through the recovery process via shell, in particular bash.

These hooks are available:

 - `PreFailover`: executed immediately before qumomf takes recovery action. Failure (non-zero exit code) of any of these processes aborts the recovery. Hint: this gives you the opportunity to abort recovery based on some internal state of your system.
 - `PostSuccessfulFailover`: executed at the end of successful recovery.
 - `PostUnsuccessfulFailover`: executed at the end of unsuccessful recovery.

Any process command that starts with "&" will be executed asynchronously, and a failure for such process is ignored.

Qumomf executes lists of commands sequentially, in order of definition.

A naive implementation might look like:

```yaml
hooks:
  shell: bash
  pre_failover:
    - "echo 'Will recover from {failureType} on {failureCluster}' >> /tmp/qumomf_recovery.log"
  post_successful_failover:
    - "echo 'Recovered from {failureType} on {failureCluster}. Set: {failureReplicaSetUUID}; Failed: {failedURI}; Successor: {successorURI}' >> /tmp/qumomf_recovery.log"
  post_unsuccessful_failover:
    - "echo 'Failed to recover from {failureType} on {failureCluster}. Set: {failureReplicaSetUUID}; Failed: {failedURI}' >> /tmp/qumomf_recovery.log"
```

### Hooks arguments and environment

Qumomf provides all hooks with failure/recovery related information, such as the UUID/URI of the failed instance, 
UUID/URI of promoted instance, type of failure, name of cluster, etc.

This information is passed independently in two ways, and you may choose to use one or both:

**Environment variables**:

  - `QUM_FAILURE_TYPE`
  - `QUM_FAILED_UUID`
  - `QUM_FAILED_URI`
  - `QUM_FAILURE_CLUSTER`
  - `QUM_FAILURE_REPLICA_SET_UUID`
  - `QUM_COUNT_FOLLOWERS`
  - `QUM_COUNT_WORKING_FOLLOWERS`
  - `QUM_COUNT_REPLICATING_FOLLOWERS`
  - `QUM_COUNT_INCONSISTENT_VSHARD_CONF`
  - `QUM_IS_SUCCESSFUL`
    
  And, if a recovery was successful:
    
  - `QUM_SUCCESSOR_UUID`
  - `QUM_SUCCESSOR_URI`

**Command line text replacement**. 

Qumomf replaces the following tokens in your hook commands:

  - `{failureType}`
  - `{failedUUID}`
  - `{failedURI}`
  - `{failureCluster}`
  - `{failureReplicaSetUUID}`
  - `{countFollowers}`
  - `{countWorkingFollowers}`
  - `{countReplicatingFollowers}`
  - `{countInconsistentVShardConf}`
  - `{isSuccessful}`

  And, if a recovery was a successful:

  - `{successorUUID}`
  - `{successorURI}`

## API

Qumomf exposes several debug endpoints:

- `/debug/metrics` - runtime and app metrics in Prometheus format,
- `/debug/health` - health check,
- `/debug/about` - the app version and build date. 

[API documentation](api/swagger.yml) for getting information about cluster states, recoveries and problems.

## Hacking

Feel free to open issues and pull requests with your ideas how to improve qumomf.

To run unit and integration tests:

```bash
make env_up
make run_tests
make env_down
```
