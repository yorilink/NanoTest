# Nano GameCluster 示例

这是一个最小可运行的游戏集群示例，用来演示 Nano 的 master、gate、game server 三类节点如何协同工作。

## 进程组成

| 进程 | 默认地址 | 作用 |
| --- | --- | --- |
| master | `127.0.0.1:34567` | 集群注册与服务发现节点 |
| game | `127.0.0.1:34680` | 承载 `GameService`，处理游戏内逻辑 |
| gate rpc | `127.0.0.1:34570` | gate 节点注册到 master 的内部 RPC 地址 |
| gate websocket | `127.0.0.1:34590` | 客户端 WebSocket 入口 |
| redis | `127.0.0.1:6379` | 存储角色、在线锁和 game server 在线人数 |

客户端连接地址：

```text
ws://127.0.0.1:34590/nano
```

## 前置条件

1. 本地已安装 Go。
2. Redis 已启动，并监听在 `127.0.0.1:6379`，或者通过 `REDIS_ADDR` 指定其他地址。
3. 在仓库根目录或任意子目录执行脚本都可以，脚本会自动切换到仓库根目录。

## 一键启动

```bash
./examples/gamecluster/start.sh
```

脚本会依次启动：

1. `master`
2. `game`
3. `gate`

`start.sh` 会先执行 `go build -o examples/gamecluster/run/gamecluster ./examples/gamecluster`，再用构建出的二进制启动三个进程。这样 PID 文件记录的是实际服务进程，关闭脚本可以可靠停止服务。

运行时文件：

| 路径 | 内容 |
| --- | --- |
| `examples/gamecluster/run/` | 进程 PID 文件 |
| `examples/gamecluster/run/gamecluster` | 启动脚本构建出的示例二进制 |
| `examples/gamecluster/logs/` | 各进程日志 |

查看日志：

```bash
tail -f examples/gamecluster/logs/master.log
tail -f examples/gamecluster/logs/game.log
tail -f examples/gamecluster/logs/gate.log
```

## 单独编译

```bash
./examples/gamecluster/build.sh
```

默认输出：

```text
examples/gamecluster/run/gamecluster
```

也可以指定输出路径：

```bash
OUTPUT=/tmp/gamecluster ./examples/gamecluster/build.sh
```

## 一键关闭

```bash
./examples/gamecluster/stop.sh
```

关闭顺序是 `gate -> game -> master`。脚本只根据 `examples/gamecluster/run/` 下的 PID 文件停止本示例启动的进程，不会按进程名批量杀进程。

## 自定义地址

可以用环境变量覆盖默认端口：

```bash
MASTER_ADDR=127.0.0.1:34567 \
GATE_RPC_ADDR=127.0.0.1:34570 \
GATE_CLIENT_ADDR=127.0.0.1:34590 \
GAME_ADDR=127.0.0.1:34680 \
REDIS_ADDR=127.0.0.1:6379 \
./examples/gamecluster/start.sh
```

常用变量：

| 环境变量 | 默认值 | 含义 |
| --- | --- | --- |
| `MASTER_ADDR` | `127.0.0.1:34567` | master RPC 地址 |
| `GATE_RPC_ADDR` | `127.0.0.1:34570` | gate 内部 RPC 地址 |
| `GATE_CLIENT_ADDR` | `127.0.0.1:34590` | gate WebSocket 客户端地址 |
| `GAME_ADDR` | `127.0.0.1:34680` | game server RPC 地址 |
| `REDIS_ADDR` | `127.0.0.1:6379` | Redis 地址 |

## 手动启动

如果不使用脚本，可以开三个终端手动启动。

终端 1：

```bash
go run ./examples/gamecluster master \
  --listen 127.0.0.1:34567
```

终端 2：

```bash
go run ./examples/gamecluster game \
  --master 127.0.0.1:34567 \
  --listen 127.0.0.1:34680 \
  --redis 127.0.0.1:6379
```

终端 3：

```bash
go run ./examples/gamecluster gate \
  --master 127.0.0.1:34567 \
  --listen 127.0.0.1:34570 \
  --gate-address 127.0.0.1:34590 \
  --redis 127.0.0.1:6379
```

## 消息入口

gate 使用 JSON 序列化，并通过 WebSocket 暴露给客户端：

```go
nano.WithIsWebsocket(true)
nano.WithWSPath("/nano")
nano.WithSerializer(json.NewSerializer())
```

示例中的服务分工：

| 服务 | 所在进程 | 职责 |
| --- | --- | --- |
| `GateService` | gate | 登录、创建角色、选择 game server、转发游戏请求 |
| `GameService` | game | 进入游戏、处理游戏逻辑、维护在线人数 |

## 常见问题

### Redis 连接失败

确认 Redis 已启动：

```bash
redis-cli -h 127.0.0.1 -p 6379 ping
```

如果 Redis 不在默认地址，启动时指定：

```bash
REDIS_ADDR=192.168.1.10:6379 ./examples/gamecluster/start.sh
```

### 端口被占用

先关闭示例：

```bash
./examples/gamecluster/stop.sh
```

如果仍然冲突，修改对应环境变量后再启动。

### 重复启动

`start.sh` 会检查 PID 文件。如果已有进程仍在运行，会跳过对应进程，避免重复启动。
