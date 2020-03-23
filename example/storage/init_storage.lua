os = require('os')
vshard = require('vshard')

local IDX_KEY = 1
local IDX_VALUE = 2

local cfg = {
    memtx_memory = 100 * 1024 * 1024,
    bucket_count = 10000,
    rebalancer_disbalance_threshold = 10,
    rebalancer_max_receiving = 100,

    -- The maximum number of checkpoints that the daemon maintains
    checkpoint_count = 6;

    -- Don't abort recovery if there is an error while reading
    -- files from the disk at server start.
    force_recovery = true;

    -- The interval between actions by the checkpoint daemon, in seconds
    checkpoint_interval = 60 * 60; -- one hour

    -- The maximal size of a single write-ahead log file
    wal_max_size = 256 * 1024 * 1024;

    wal_mode = "write";

    memtx_min_tuple_size = 16;
    memtx_max_tuple_size = 128 * 1024 * 1024; -- 128Mb

    readahead = 16320;

    sharding = {
        ['7432f072-c00b-4498-b1a6-6d9547a8a150'] = { -- replicaset #1
            replicas = {
                ['294e7310-13f0-4690-b136-169599e87ba0'] = {
                    uri = 'qumomf:qumomf@qumomf_1_m.ddk:3301',
                    name = 'qumomf_1_m',
                    master = true
                },
                ['cd1095d1-1e73-4ceb-8e2f-6ebdc7838cb1'] = {
                    uri = 'qumomf:qumomf@qumomf_1_s.ddk:3301',
                    name = 'qumomf_1_s'
                }
            },
        }, -- replicaset #1
        ['5065fb5f-5f40-498e-af79-43887ba3d1ec'] = { -- replicaset #2
            replicas = {
                ['f3ef657e-eb9a-4730-b420-7ea78d52797d'] = {
                    uri = 'qumomf:qumomf@qumomf_2_m.ddk:3301',
                    name = 'qumomf_2_m',
                    master = true
                },
                ['7d64dd00-161e-4c99-8b3c-d3c4635e18d2'] = {
                    uri = 'qumomf:qumomf@qumomf_2_s.ddk:3301',
                    name = 'qumomf_2_s'
                }
            },
        }, -- replicaset #2
    }, -- sharding
}

local UUID = os.getenv("STORAGE_UUID")

cfg.listen = 3301
vshard.storage.cfg(cfg, UUID)

box.once("init", function()
    box.schema.user.create('qumomf', { password = 'qumomf', if_not_exists = true })
    box.schema.user.grant('qumomf', 'read,write,create,execute', 'universe')

    local space = box.schema.create_space("qumomf", {
        if_not_exists = true,
    })

    space:create_index('key', {
        type = 'TREE',
        if_not_exists = true,
        parts = {
            IDX_KEY,
            'string',
        },
        unique = true,
    })
end)

function qumomf_change_master(shard_uuid, old_master_uuid, new_master_uuid)
    local replicas = cfg.sharding[shard_uuid].replicas
    replicas[old_master_uuid].master = false
    replicas[new_master_uuid].master = true
    vshard.storage.cfg(cfg, os.getenv('STORAGE_UUID'))
end

dofile('/etc/tarantool/instances.enabled/qumomf/storage/storage.lua')