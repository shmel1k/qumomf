FROM registry.city-srv.ru/a.petrukhin/tarantool-docker:master
COPY init_storage.lua /etc/tarantool/instances.enabled/init_storage.lua
COPY storage.lua /etc/tarantool/instances.enabled/opc/storage/storage.lua

CMD ["tarantool", "/etc/tarantool/instances.enabled/init_storage.lua"]
