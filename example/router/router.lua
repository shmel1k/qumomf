vshard = require('vshard')

local DEFAULT_TIMEOUT = 1

local OP_GET = 'qumomf_get'
local OP_SET = 'qumomf_set'

function qumomf_get(key)
    local bucket_id = vshard.router.bucket_id(key)
    local netbox, err = vshard.router.route(bucket_id)
    if err ~= nil then
        error(err)
    end

    local result, err = netbox:callbre(OP_GET, {key}, {
        timeout = DEFAULT_TIMEOUT,
    })
    if err ~= nil then
        error(err)
    end
    return result
end

function qumomf_set(key, value)
    local bucket_id = vshard.router.bucket_id(key)
    local netbox, err = vshard.router.route(bucket_id)
    if err ~= nil then
        error(err)
    end

    local result, err = netbox:callrw(OP_SET, { key, value }, {
        timeout = DEFAULT_TIMEOUT,
    })
    if err ~= nil then
        error(err)
    end

    return result
end