#!/bin/bash

# 将宿主机 /rtfd 目录映射到容器内，数据和配置文件皆放到 /rtfd 目录下！

docker run -tdi --name rtfd --restart=always --net=host -v /rtfd:/rtfd staugur/rtfd
