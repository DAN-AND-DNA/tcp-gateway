# gateway
- This gateway can forward TCP messages to the backend HTTP service.
- HTTP service can push messages to TCP connections.
- Production environment validation (millions of daily active users)

## compile
```bash
make        # linux dev
build.sh    # linux dev

make release       # linux release
build.sh release   # linux release
```



## usage
```bash
# Start and specify the Redis address  
./gateway -discovery_redis_address=127.0.0.1:6379

# Start and specify the ports  
./gateway -node_info_private_http_port=18081 -node_info_public_tcp_port=18001

# Run without service discovery
./gateway -discovery_type=none

# Other options
./gateway -h
```

## TODO List
- ~~Remove dependency on cos (删除依赖cos)~~
- Multi-platform API plugin (多平台api插件)
- Alert plugin (告警插件)
- TCP plugin (tcp插件)
- HTTP plugin (http插件)
- Provide examples (提供例子)

## performance
![机器](./img/20250528-112705.jpg)
![机器](./img/20250528-112712.jpg)
![机器](./img/20250528-112914.jpg)