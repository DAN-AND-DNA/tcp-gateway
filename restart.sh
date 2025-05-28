#!/bin/bash
set -ex
BASE=$(dirname "$(realpath "$0")")
ps -ef | awk '/gateway/ && !/pushgateway/ {print $2}'| xargs -r kill -2
sleep 2
export BUILD_ID=dontKillMe # 防止Jenkins清理
nohup $BASE/gateway -discovery_redis_address=127.0.0.1:6379 -node_info_private_http_port=28081 -node_info_public_tcp_port=28001 > $BASE/gateway1.log 2>&1 < /dev/null &
nohup $BASE/gateway -discovery_redis_address=127.0.0.1:6379 -node_info_private_http_port=28082 -node_info_public_tcp_port=28002 > $BASE/gateway2.log 2>&1 < /dev/null &