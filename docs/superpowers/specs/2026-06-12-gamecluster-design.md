# Gamecluster 示例设计

## 目标

在 `examples/gamecluster` 中实现一个聚焦的 Nano 集群示例，用来展示常见游戏服务器结构：

- 客户端只连接 gate；
- gate 负责账号认证、查询 Redis 角色摘要、选择 gameserver；
- gameserver 拥有在线玩家状态并处理游戏消息；
- Nano cluster 将 `GameService.*` 消息从 gate 转发到选定 gameserver。

第一版只验证登录、创建角色、重连、路由绑定和基础负载均衡流程。不修改 Nano 核心框架能力。

## 进程类型

### Master

master 使用 Nano 现有 cluster master 能力，负责服务注册、服务发现、心跳和节点下线通知。master 不处理账号、角色或游戏业务。

### Gate

gate 接收客户端 WebSocket 连接，注册 `GateService`。它是客户端唯一入口。

职责：

- 校验登录 token，得到 `accountId`；
- 从 Redis 读写角色摘要；
- 账号没有角色时告诉客户端需要创建角色；
- 根据客户端传来的名字创建角色；
- 按重连优先或在线人数负载选择 gameserver；
- 将 `session.UID` 绑定为 `playerId`；
- 在 session 上缓存 `accountId`、`playerId`、`name`、`gameServerAddr`；
- 用 `session.Router().Bind("GameService", gameServerAddr)` 将该客户端的 `GameService` 固定到选定 gameserver；
- 调用 `GameService.Enter`，成功后再返回登录或建角成功。

gate 只保存路由和摘要信息，不拥有完整玩家状态。

### Gameserver

每个 gameserver 注册 `GameService`。

职责：

- 接收 gate 调用的 `GameService.Enter`；
- 维护本进程在线玩家表；
- 拥有在线游戏状态；
- 增减本服在 Redis 中的在线人数；
- 处理后续 `GameService.*` 游戏消息；
- Nano session 断开时释放在线状态。

## 登录认证

示例使用可替换的 token verifier。第一版支持 demo token：

```text
demo:{accountId}
```

例如 `demo:10001` 解析出账号 ID `10001`。

token verifier 应隐藏在一个小接口后面。以后替换成 JWT、平台登录或其它账号系统时，不影响角色路由和 gameserver 进入逻辑。

## 客户端消息流

### 登录但没有角色

1. 客户端请求 `GateService.Login`，参数包含 token。
2. gate 校验 token，得到 `accountId`。
3. gate 读取 `gamecluster:account:{accountId}:player`。
4. 如果没有角色，gate 返回 `needCreateRole=true`。
5. gate 不绑定 `playerId`，也不进入 gameserver。

### 创建角色

1. 客户端请求 `GateService.CreateRole`，参数包含 token 和名字。
2. gate 再次校验 token。
3. gate 检查账号是否已经有角色。
4. 如果角色已存在，gate 返回已有角色，并继续走进入 gameserver 流程，不创建第二个角色。
5. 如果角色不存在，gate 使用 `INCR gamecluster:player_id` 生成全局唯一 `playerId`。
6. gate 选择当前可用 gameserver 中在线人数最少的节点。
7. gate 将账号和角色摘要写入 Redis。
8. gate 进入选定 gameserver。

### 已有角色登录或重连

1. gate 读取角色摘要和上次绑定的 `gameServerAddr`。
2. 如果该地址仍然是当前 Nano cluster 中提供 `GameService` 的成员，gate 优先选择它。这样 gameserver 有机会按重连处理，并复用内存里的玩家状态。
3. 如果旧 gameserver 不可用，或没有旧地址，gate 选择 Redis 在线人数最低的可用 gameserver。
4. 如果选定 gameserver 发生变化，gate 更新 Redis 中的角色摘要。
5. gate 绑定 session：

```go
s.Bind(playerId)
s.Set("accountId", accountId)
s.Set("playerId", playerId)
s.Set("name", name)
s.Set("gameServerAddr", gameServerAddr)
s.Router().Bind("GameService", gameServerAddr)
```

6. gate 调用 `s.RPC("GameService.Enter", enterReq)`。
7. gameserver 如果本地在线表已有该玩家，按重连处理；否则从 Redis 恢复最小 profile，按新登录处理。
8. `Enter` 成功后，gate 回复客户端成功。

进入成功后，客户端直接发送 `GameService.*` 游戏请求。gate 不手写业务消息转发，交给 Nano cluster 根据 session 路由绑定完成转发。

## Redis 数据

第一版 Redis 只保存账号、角色摘要、在线路由和负载计数。

### 玩家 ID 计数器

```text
gamecluster:player_id
```

使用 `INCR` 生成全局唯一 `playerId`。

### 账号到角色摘要

```text
gamecluster:account:{accountId}:player
```

Hash 字段：

```text
playerId
name
gameServerAddr
```

### 角色摘要

```text
gamecluster:player:{playerId}:profile
```

Hash 字段：

```text
accountId
name
gameServerAddr
```

### 账号在线锁

```text
gamecluster:online:account:{accountId}
```

Hash 字段：

```text
gateAddr
sessionId
playerId
gameServerAddr
loginAt
```

gameserver 进入成功后，gate 写入在线锁；gate session 关闭时删除。在线锁应带 TTL，例如 60 秒，用来兜底 gate 崩溃。后续生产版本可以在心跳中刷新 TTL，或增加跨 gate 踢线 RPC。

### Gameserver 在线人数

```text
gamecluster:gameserver:{serviceAddr}:online_count
```

玩家首次进入本服成功后，gameserver 增加计数；玩家离开时减少计数。重连到已经在线的玩家不能重复增加计数。

## 单连接规则

一个账号只允许一个活跃连接。

第一版示例先在单 gate 进程内清楚执行这个规则：gate 维护内存 `accountId -> session` 映射。新连接登录成功后，旧本地 session 被关闭或失效。

多 gate 场景下，Redis 在线锁会记录旧 gate 和旧 session。第一版可以在新连接进入 gameserver 成功后覆盖旧远端在线锁。跨 gate 强制踢线是后续功能，不放进第一版。

## Gameserver 选择

选择顺序：

1. 如果角色摘要中的 `gameServerAddr` 仍然是提供 `GameService` 的可用 Nano cluster 成员，优先选择它。
2. 否则选择 Redis 在线人数最低的可用 gameserver。
3. 如果没有任何可用 gameserver，返回服务不可用错误。

gate 不能盲信 Redis 中的旧地址，必须和当前提供 `GameService` 的 cluster 成员做比较。

## 错误处理

- token 无效：`GateService.Login` 和 `GateService.CreateRole` 返回 `code=401`。
- 登录时没有角色：返回 `needCreateRole=true`，不绑定 `playerId`。
- 重复创建角色：返回已有角色，并继续进入 gameserver。
- 没有可用 gameserver：返回 `code=503`，不能假装登录成功。
- Redis 不可用：返回 `code=500`，不使用猜测的本地状态降级。
- 旧 gameserver 不可用：选择新的最低在线人数 gameserver，并更新 Redis。
- `GameService.Enter` 失败：删除本 session 的 `GameService` 路由绑定，向客户端返回进入失败。
- gate 崩溃：Redis 在线锁 TTL 到期后允许重新登录。
- gameserver 崩溃：Nano master 摘除该成员，后续登录选择其它 gameserver。

## 测试范围

重点测试：

- token verifier：合法 `demo:{accountId}` 和非法 token；
- Redis repository：创建角色、查询角色、更新 gameserver 地址、在线锁、在线人数；
- gameserver selector：优先可用旧服、旧服不可用时回退、选择最低在线人数、没有 gameserver 时报错；
- gate service：无角色登录、创建角色、重复创建、已有角色登录；
- gameserver service：首次 Enter、重连 Enter、断线清理在线人数。

集成测试可以参考 `examples/cluster`：启动 master、gate、两个 gameserver，然后用客户端跑通登录、创建角色、重连，以及某个 gameserver 退出后回退到其它 gameserver。

## 明确不做

第一版不实现：

- 完整 JWT、OAuth 或平台登录；
- 跨 gate 强制踢旧连接 RPC；
- 完整玩家状态、背包、地图、战斗或持久化；
- 在线过程中自动迁服；
- CPU、内存、房间数或地图分片等负载指标。

这些都是合理的后续扩展，但会干扰第一版目标。第一版重点是展示清楚的 gate 到 gameserver 流程：Redis 角色摘要、重连路由、单连接规则和 Nano cluster 消息转发。
