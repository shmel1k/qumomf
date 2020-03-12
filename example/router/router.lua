platform = require('platform')
vshard = require('vshard')
vshard = require('netbox')

local DEFAULT_TIMEOUT = 1

local MODE_READ = 'read'
local MODE_WRITE = 'write'

local OP_GET = 'qumomf_get'
local OP_SET = 'qumomf_set'

local function qumomf_get(key)
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

local function qumomf_set(key, value)
    local bucket_id = vshard.router.bucket_id(key)
    return vshard.router.call(bucket_id, MODE_WRITE, OP_SET, {key, value}, {
        timeout = DEFAULT_TIMEOUT,
    })
end

platform.init({
    functions = {
        qumomf_get = platform.wrap_func('qumomf_get', qumomf_get),
        qumomf_set = platform.wrap_func('qumomf_set', qumomf_set),
    },
})