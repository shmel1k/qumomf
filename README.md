# Qumomf

Qumomf is a Tarantool vshard high availability tool which supports discovery and recovery.

## How to add a new cluster

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

Start qumomf, and it will discover all defined clusters.

## Test

```bash
# Unit & Integration tests
make env_up
make run_tests
make env_down
```
