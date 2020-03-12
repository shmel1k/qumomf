FROM registry.city-srv.ru/a.petrukhin/tarantool-docker:master
COPY init_router.lua /etc/tarantool/instances.enabled/init_router.lua
COPY router.lua /etc/tarantool/instances.enabled/opc/router/router.lua
CMD ["tarantool", "/etc/tarantool/instances.enabled/init_router.lua"]
