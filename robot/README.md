# Nano Robot

`robot` 是用于 `examples/gamecluster` 的登录机器人和压测入口。它走真实
WebSocket 客户端链路，默认请求 `GateService.Login`，首次登录返回
`needCreateRole` 时会自动调用 `GateService.CreateRole`。

## 构建

```bash
go build -o bin/robot ./robot
```

## 示例

单个机器人登录：

```bash
./bin/robot -addr 127.0.0.1:34590
```

一万个账号并发登录：

```bash
./bin/robot -addr 127.0.0.1:34590 -count 10000 -concurrency 10000
```

更稳的阶梯压测可以降低并发 worker 数：

```bash
./bin/robot -addr 127.0.0.1:34590 -count 10000 -concurrency 1000
```

账号 token 使用 demo 规则生成：`demo:<accountID>`。可以用
`-start-account` 控制起始账号。

## 日志

默认日志目录是 `robot/logs`，每次运行会生成一个独立日志文件：

```text
robot/logs/robot-YYYYMMDD-HHMMSS-PID.log
```

统计信息会同时输出到终端和日志文件。连接、登录、建角、插件执行失败时，
日志会记录 `accountID`、失败阶段和错误原因。可以用 `-log-dir` 修改目录：

```bash
./bin/robot -log-dir /tmp/nano-robot-logs
```

## 登录后协议插件

登录成功后会执行 `Plugin` 接口。后续要加自动发协议，只需要实现插件并在
`buildPlugins` 中注册：

```go
type Plugin interface {
	Name() string
	Run(ctx context.Context, c *Client, accountID int64, login *GateResponse) error
}
```

插件内可以复用 `Client.Request` 发送任意 JSON 协议。
