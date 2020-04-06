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

Add Lua procedure on all storages or adapt it for your environment:

```lua
function qumomf_change_master(shard_uuid, old_master_uuid, new_master_uuid)
    local replicas = cfg.sharding[shard_uuid].replicas
    replicas[old_master_uuid].master = false
    replicas[new_master_uuid].master = true
    vshard.storage.cfg(cfg, os.getenv('STORAGE_UUID'))
end
```

On routers:

```lua
function qumomf_change_master(shard_uuid, old_master_uuid, new_master_uuid)
    local replicas = cfg.sharding[shard_uuid].replicas
    replicas[old_master_uuid].master = false
    replicas[new_master_uuid].master = true
    vshard.router.cfg(cfg)
end
```

`cfg` is a local variable contains the configuration of the replica sets. 
For a sample configuration, see [qumomf example](/example) or [Tarantool documentation](https://www.tarantool.io/en/doc/1.10/reference/reference_rock/vshard/vshard_quick/#vshard-config-cluster-example).

Start or restart qumomf and the orchestrator will discover all configured clusters.

## Test

```bash
# Unit & Integration tests
make env_up
make run_tests
make env_down
```
