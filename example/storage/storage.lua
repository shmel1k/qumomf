require('strict').on()
platform = require('platform')

local function qumomf_set(key, value)
    box.space.qumomf:put({key, value})
    return {}
end

local function qumomf_get(key)
    local tuple = box.space.qumomf:select(key)
    if #tuple == 0 then
        return nil
    end
    tuple = tuple[1]

    return tuple
end

platform.init({
    functions = {
        qumomf_get = platform.wrap_func('qumomf_get', qumomf_get),
        qumomf_set = platform.wrap_func('qumomf_set', qumomf_set),
    },
})
