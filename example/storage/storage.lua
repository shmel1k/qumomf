require('strict').on()
os = require('os')

local IDX_KEY = 1
local IDX_VALUE = 2

function qumomf_set(key, value, expiration_ts)
    box.space.qumomf:insert({ key, value, 0 })
    return {}
end

function qumomf_get(key)
    local tuple = box.space.qumomf:select(key)
    if #tuple == 0 then
        return nil
    end
    tuple = tuple[1]

    return tuple[IDX_VALUE]
end
