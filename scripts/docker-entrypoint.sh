#!/bin/sh
# 容器启动入口：
# 1) 确保 rtfd 配置文件存在——它位于 base_dir 内（默认 /rtfd/rtfd.cfg），
#    便于把配置与数据（docs/db/caddy）收敛到同一个数据卷；
# 2) 当 /rtfd 被挂载为卷（尤其 bind mount）导致镜像内自带配置被遮住时，
#    用内置默认模板补生成一份（支持 RTFD_API_SERVER_URL / RTFD_CADDY_DN 预填）；
# 3) 交由 CMD 启动 supervisord（默认 `supervisord`）。
set -e

cfg="${RTFD_CFG:-/rtfd/rtfd.cfg}"
mkdir -p "$(dirname "$cfg")" 2>/dev/null || true

if [ ! -f "$cfg" ]; then
    if [ -f /rtfd.cfg ]; then
        # 兼容旧布局：配置曾在镜像根 /rtfd.cfg，迁移到 base_dir 内
        echo "[entrypoint] migrating legacy config /rtfd.cfg -> $cfg"
        cp -f /rtfd.cfg "$cfg"
    else
        echo "[entrypoint] rtfd config not found, generating default: $cfg"
        rtfd -c "$cfg" --init || true
    fi
fi

exec "$@"
