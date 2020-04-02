FROM tarantool/tarantool:2.3.1

COPY init_router.lua /etc/tarantool/instances.enabled/init_router.lua
COPY router.lua /etc/tarantool/instances.enabled/qumomf/router/router.lua
CMD ["tarantool", "/etc/tarantool/instances.enabled/init_router.lua"]
