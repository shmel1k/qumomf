# Qumomf

Qumomf is a Tarantool vshard high availability tool which supports discovery and recovery.

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
        uuid: 'my_cluster_router_1'
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
        uuid: 'my_cluster_router_1'
```

For a sample vshard configuration, 
see [qumomf example](/example) or [Tarantool documentation](https://www.tarantool.io/en/doc/1.10/reference/reference_rock/vshard/vshard_quick/#vshard-config-cluster-example).

Start qumomf, and it will discover all clusters defined in the configuration.

## Topology recovery

Just now qumomf supports only automated master recovery.
It is a configurable option and can be disabled completely or for a cluster via configuration.

Master election supports two modes:

1. `delay` - naive and simple elector which finds alive replica last communicated to the failed master (received data or heartbeat signal).
2. `smart` - elector tries to involve as many metrics as can:
  - vshard configuration consistency (prefer replica which has the same configuration as master), 
  - which upstream status did replica have before the crash,
  - how replica is far from master comparing LSN to master LSN,
  - last time when replica received data or heartbeat signal from master. 

Election mode might be configured for each cluster.

## Hacking

Feel free to open issues and pull requests with your ideas how to improve qumomf.

To run unit and integration tests:

```bash
make env_up
make run_tests
make env_down
```
