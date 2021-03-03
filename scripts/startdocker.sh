#!/bin/bash

# 将宿主机 /rtfd 数据目录映射到容器内，配置文件固定为 /rtfd.cfg

docker run -tdi --name rtfd --restart=always --net=host \
    -v /rtfd:/rtfd -v rtfd.cfg:/rtfd.cfg staugur/rtfd
