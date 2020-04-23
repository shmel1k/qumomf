FROM tarantool/tarantool:2.3.1

COPY --from=trajano/alpine-libfaketime /faketime.so /lib/faketime.so
ENV LD_PRELOAD=/lib/faketime.so

COPY init_storage.lua /etc/tarantool/instances.enabled/init_storage.lua
COPY storage.lua /etc/tarantool/instances.enabled/qumomf/storage/storage.lua
CMD ["tarantool", "/etc/tarantool/instances.enabled/init_storage.lua"]
