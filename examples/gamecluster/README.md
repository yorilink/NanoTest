# Nano gamecluster example

这个示例演示一个基础游戏集群：

- 客户端只连接 `gate`；
- `gate` 校验 demo token，查询或创建角色摘要，并选择 `gameserver`；
- `gameserver` 维护在线玩家状态；
- 客户端后续直接请求 `GameService.*`，由 Nano cluster 转发到绑定的 gameserver。

## 依赖

示例运行时默认连接 Redis：

```shell
redis-server
```

默认地址是 `127.0.0.1:6379`，也可以通过 `--redis` 指定。

当前 `store.RedisRepository` 是示例级最小 RESP 客户端，只实现本示例需要的 Redis 命令。生产环境应替换为成熟 Redis 客户端，并用 Lua 或事务保证建角写入的原子性。

## 运行

```shell
cd examples/gamecluster
go build

./gamecluster master
./gamecluster game --listen "127.0.0.1:34680"
./gamecluster game --listen "127.0.0.1:34681"
./gamecluster gate --listen "127.0.0.1:34570" --gate-address "127.0.0.1:34590"
```

WebSocket 客户端连接：

```text
ws://127.0.0.1:34590/nano
```

登录 token 使用 demo 格式：

```text
demo:10001
```

## 客户端路由

登录：

```text
GateService.Login
```

请求：

```json
{"token":"demo:10001"}
```

如果没有角色，会返回 `needCreateRole=true`。

创建角色：

```text
GateService.CreateRole
```

请求：

```json
{"token":"demo:10001","name":"Alice"}
```

进入成功后，gameserver 会推送：

```text
GameService.Entered
```

验证后续游戏路由：

```text
GameService.Ping
```

请求：

```json
{"content":"hello"}
```

这个请求会被 Nano cluster 转发到 gate 绑定的 gameserver。
