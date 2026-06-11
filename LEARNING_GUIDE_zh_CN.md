# Nano 学习导读

本导读用于按顺序学习 `lonng/nano` 服务器框架。仓库已经下载到：

```bash
/home/dbn/nano
```

当前检出的版本：

```text
dbf22c7 update Community info in README (#117)
```

## 0. 先认识 Nano

Nano 是一个 Go 写的轻量级实时服务器网络框架，主要面向游戏、社交游戏、移动游戏和高实时 Web 应用。它的核心模型是：

- 一个 Nano 应用由多个 `Component` 组成。
- 每个 `Component` 里符合签名的方法会被注册为 `Handler`。
- 客户端通过 `route` 调用服务端 Handler，例如 `room.message` 或 `Room.Message`。
- 每个连接对应一个 `session.Session`。
- 服务端可通过 `Response` 回复请求，也可通过 `Push` 主动推送。
- 多个 session 可加入 `Group`，用于房间广播。

建议先读：

- `README.md`
- `docs/get_started_zh_CN.md`

## 1. 跑通第一个聊天室

进入聊天示例：

```bash
cd /home/dbn/nano/examples/demo/chat
go run main.go
```

浏览器打开：

```text
http://localhost:3250/web/
```

重点看：

- `examples/demo/chat/main.go`
- `examples/demo/chat/web/index.html`
- `examples/demo/chat/web/protocol.js`
- `examples/demo/chat/web/starx-wsclient.js`

先理解这几个点：

- `RoomManager` 是一个组件。
- `Join` 和 `Message` 是客户端可以调用的 Handler。
- `components.Register(...)` 把组件注册进 Nano。
- `nano.Listen(":3250", ...)` 启动服务器。
- `nano.WithIsWebsocket(true)` 开启 WebSocket。
- `nano.WithSerializer(json.NewSerializer())` 使用 JSON 编解码。
- `group.Broadcast(...)` 给房间成员广播消息。

## 2. 学 Component 和 Handler

重点文件：

- `component/base.go`
- `component/component.go`
- `component/hub.go`
- `component/method.go`
- `component/options.go`
- `docs/get_started_zh_CN.md`

你需要掌握：

- 如何定义组件结构体。
- 为什么组件通常嵌入 `component.Base`。
- 生命周期方法：`Init`、`AfterInit`、`BeforeShutdown`、`Shutdown`。
- Handler 的合法签名：

```go
func (c *DemoComponent) DemoHandler(s *session.Session, payload *SomePayload) error
func (c *DemoComponent) DemoHandler(s *session.Session, raw []byte) error
```

实践任务：

- 在聊天示例里新增一个 Handler，例如 `Ping`。
- 客户端调用 `room.ping`，服务端返回当前时间或固定字符串。

## 3. 学 Session 和 Group

重点文件：

- `session/session.go`
- `session/lifetime.go`
- `group.go`
- `examples/demo/chat/main.go`

你需要掌握：

- `s.ID()` 是连接级 ID。
- `s.Bind(uid)` 可把业务用户 ID 绑定到 session。
- `s.Set(key, value)`、`s.Value(key)`、`s.HasKey(key)` 可保存连接上下文。
- `session.Lifetime.OnClosed(...)` 可处理断线清理。
- `nano.NewGroup(...)` 创建分组。
- `group.Add(s)`、`group.Leave(s)`、`group.Broadcast(...)` 实现房间广播。

实践任务：

- 给聊天室增加多个房间。 
- `Join` 时从客户端传房间 ID。
- `Message` 只广播给当前房间。

我虽然每写这个 但是我说一下思路 修改demo里面的join,在上行的协议里带过来，然后做校验，再加入指定的房间
再广播给这个房间，而不是只加入到testRoomID这个里面

## 4. 学请求、响应、通知、推送

重点文档：

- `docs/get_started_zh_CN.md`
- `docs/communication_protocol_zh_CN.md`

Nano 消息模型：

- `Request`：客户端请求服务端，服务端需要返回 `Response`。
- `Response`：服务端对请求的响应。
- `Notify`：客户端通知服务端，不需要响应。
- `Push`：服务端主动推给客户端。

服务端常用写法：

```go
return s.Response(payload)
s.Push("onEvent", payload)
return group.Broadcast("onMessage", payload)
```

实践任务：

- 把 `Join` 做成 request-response。
- 把 `Message` 做成 notify。
- 服务端收到消息后用 push/broadcast 推送给其他人。

## 5. 学序列化：JSON 和 Protobuf

重点文件：

- `serialize/serializer.go`
- `serialize/json/json.go`
- `serialize/protobuf/protobuf.go`
- `benchmark/testdata/test.proto`

聊天示例使用 JSON：

```go
nano.WithSerializer(json.NewSerializer())
```

真实项目更常用 Protobuf，因为体积更小、结构更稳定、跨语言更明确。

实践任务：

- 先继续用 JSON 完成业务。
- 熟悉后再把一个消息结构改成 Protobuf。
- 阅读 `docs/communication_protocol_zh_CN.md` 理解 Nano 外层包和内部 message 的关系。

## 6. 学 Pipeline 中间件

重点文件：

- `pipeline/pipeline.go`
- `examples/demo/chat/main.go`

聊天示例中统计流量的逻辑就是 Pipeline：

```go
pip := pipeline.New()
pip.Outbound().PushBack(stats.outbound)
pip.Inbound().PushBack(stats.inbound)
nano.WithPipeline(pip)
```

你可以把 Pipeline 理解成消息进出服务器时的中间处理链。

实践任务：

- 在 inbound pipeline 打印 route、数据长度或 session ID。
- 在 outbound pipeline 统计推送消息数量。

## 7. 学定时器和主逻辑线程调度

重点文件：

- `scheduler/timer.go`
- `scheduler/scheduler.go`
- `examples/demo/chat/main.go`

Nano 里可以用 `scheduler.NewTimer(...)` 做周期任务。README 还展示了一个典型模式：慢任务放到 goroutine，结果回到 Nano 主逻辑调度里处理。

实践任务：

- 每 10 秒打印一次在线人数。
- 模拟数据库查询：goroutine 中 sleep，然后调用 `nano.Invoke(...)` 回到逻辑调度处理结果。

## 8. 学路由压缩

重点文档：

- `docs/route_compression_zh_CN.md`
- `docs/communication_protocol_zh_CN.md`

路由压缩用于把较长的字符串 route 映射为短数字，减少网络包体积。适合消息频率很高的游戏和实时应用。

实践任务：

- 先不用路由压缩，把功能跑通。
- 再给常用 route 配字典。
- 对比压缩前后的消息大小。

## 9. 学集群模式

重点文件：

- `cluster/`
- `examples/cluster/README.md`
- `examples/cluster/main.go`
- `examples/cluster/master/`
- `examples/cluster/gate/`
- `examples/cluster/chat/`

运行方式：

```bash
cd /home/dbn/nano/examples/cluster
go build
./cluster master
./cluster chat --listen "127.0.0.1:34580"
./cluster gate --listen "127.0.0.1:34570" --gate-address "127.0.0.1:34590"
```

建议理解：

- `master` 负责服务发现和成员管理。
- `gate` 面向客户端连接。
- `chat` 是后端业务服务。
- 单机模式先学透，再看集群。

## 10. 学自定义路由

重点文件：

- `examples/customerroute/README.md`
- `examples/customerroute/main.go`
- `examples/customerroute/onegate/`
- `examples/customerroute/onemaster/`
- `examples/customerroute/tworoom/`
- `options.go` 中的 `WithCustomerRemoteServiceRoute`

这部分适合你已经理解集群之后再看。它解决的是：请求进入集群后，如何按自定义规则分发到后端服务。

## 推荐学习顺序

1. `README.md`
2. `docs/get_started_zh_CN.md`
3. 跑 `examples/demo/chat`
4. 改聊天示例：新增 Handler、房间、在线人数
5. 读 `session/`、`group.go`、`component/`
6. 读 `pipeline/`、`scheduler/`
7. 读 `docs/communication_protocol_zh_CN.md`
8. 读 `docs/route_compression_zh_CN.md`
9. 跑 `examples/cluster`
10. 看 `examples/customerroute`

## 建议做的小项目

按这个顺序做一个小型实时房间服务：

1. 用户连接 WebSocket。
2. 用户登录并绑定 uid。
3. 用户加入指定房间。
4. 房间内发送聊天消息。
5. 用户断线后自动离开房间。
6. 服务端每 10 秒广播房间在线人数。
7. 增加一个私聊接口。
8. 增加消息日志 pipeline。
9. 改成 Protobuf。
10. 最后拆成 gate + room 后端服务。

## 常用命令

运行全部测试：

```bash
cd /home/dbn/nano
go test ./...
```

运行聊天 demo：

```bash
cd /home/dbn/nano/examples/demo/chat
go run main.go
```

运行集群 demo：

```bash
cd /home/dbn/nano/examples/cluster
go build
./cluster master
./cluster chat --listen "127.0.0.1:34580"
./cluster gate --listen "127.0.0.1:34570" --gate-address "127.0.0.1:34590"
```
