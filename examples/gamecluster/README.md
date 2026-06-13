 func Listen(addr string, opts ...Option) 里的 opts ...Option 是 Go 的可变参数，意思是调用 nano.Listen
  时可以传 0 个、1 个或多个配置项。

  在 nano 里，Option 本质上是：

  type Option func(*cluster.Options)

  也就是说，每个 Option 都是一个函数，用来修改 nano 节点启动配置。

  对你的 gamecluster 项目来说，addr 是当前进程的 RPC 服务地址，opts 决定这个进程是 master、gate 还是
  gameserver，以及它注册哪些 component、是否开启 WebSocket、怎么连 master 等。

  例如 master：

  nano.Listen(listen,
      nano.WithMaster(),
      nano.WithSerializer(json.NewSerializer()),
      nano.WithDebugMode(),
  )

  含义：

  - listen：master 的 RPC 地址，比如 127.0.0.1:34567
  - WithMaster()：这个节点是 cluster master
  - WithSerializer(...)：消息序列化用 JSON
  - WithDebugMode()：打开调试日志

  gate：

  nano.Listen(listen,
      nano.WithAdvertiseAddr(masterAddr),
      nano.WithClientAddr(gateAddr),
      nano.WithComponents(gate.Services),
      nano.WithSerializer(json.NewSerializer()),
      nano.WithIsWebsocket(true),
      nano.WithWSPath("/nano"),
      nano.WithCheckOriginFunc(func(_ *http.Request) bool { return true }),
      nano.WithCustomerRemoteServiceRoute(gate.RouteService().Route),
      nano.WithDebugMode(),
      nano.WithNodeId(2),
  )

  对你的项目最关键的是：

  - WithAdvertiseAddr(masterAddr)：告诉 gate 去哪里注册到 master
  - WithClientAddr(gateAddr)：客户端 WebSocket 连接地址，比如 127.0.0.1:34590
  - WithComponents(gate.Services)：注册 GateService
  - WithIsWebsocket(true)：gate 接收 WebSocket 客户端
  - WithWSPath("/nano")：客户端连接路径 /nano
  - WithCustomerRemoteServiceRoute(...)：决定 GameService 消息转到哪个 gameserver
  - WithNodeId(2)：设置 sessionId 雪花节点 ID，多 gate 时不能重复

  gameserver：

  nano.Listen(listen,
      nano.WithAdvertiseAddr(masterAddr),
      nano.WithComponents(game.Services),
      nano.WithSerializer(json.NewSerializer()),
      nano.WithDebugMode(),
  )

  含义：

  - listen：gameserver 的 RPC 地址，比如 127.0.0.1:34680
  - WithAdvertiseAddr(masterAddr)：注册到 master
  - WithComponents(game.Services)：注册 GameService

  简单说：

  Listen(addr, opts...)

  就是：

  启动一个 nano 节点，addr 是节点内部 RPC 地址，opts 是这个节点的各种配置开关。

  对你的游戏集群：

  - master：只需要 WithMaster
  - gate：需要 WebSocket、GateService、自定义 GameService 路由
  - gameserver：需要 GameService，通常不需要客户端 WebSocket 端口