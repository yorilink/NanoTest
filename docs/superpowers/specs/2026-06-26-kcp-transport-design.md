# KCP Client Transport Design

## Goal

Add KCP as an optional client transport for gate nodes. The first version keeps
the existing nano packet, message, session, pipeline, and cluster RPC behavior
unchanged. KCP only replaces the client-side socket transport.

## Scope

- Add a KCP listener address to `cluster.Options`.
- Add public options in the root `nano` package.
- Accept KCP sessions and pass them into the existing `LocalHandler.handle`
  method as `net.Conn`.
- Keep TCP and WebSocket behavior unchanged.
- Do not modify protobuf files, business components, serializers, or the
  internal cluster gRPC path.

## Architecture

The existing client path already depends on `net.Conn` at the handler boundary.
`LocalHandler.handle` creates an `agent`, starts the write goroutine, decodes
nano packets, performs handshake state transitions, and dispatches messages.

KCP will be added as another listener owned by `Node`:

```text
client KCP session -> kcp listener -> LocalHandler.handle(net.Conn)
```

This keeps ownership deterministic: the gate node owns the KCP session, the
agent owns read/write lifecycle, and business services continue to own game
state through sessions and schedulers.

## Configuration

The first implementation exposes:

- `WithKCPAddr(addr string)` to enable KCP.
- `WithKCPConfig(config cluster.KCPConfig)` for tuning.

`KCPConfig` includes nodelay parameters, MTU, send and receive windows, DSCP,
and socket buffer sizes. Zero values use conservative defaults in the listener.

## Error Handling

Listener startup failures are fatal, matching the current TCP and WebSocket
client listener behavior. Per-session accept errors are logged and do not stop
the listener. Session close and remote session cleanup continue through the
existing `agent.Close` and `SessionClosed` flow.

## Testing

Unit coverage should verify KCP defaults and option wiring. Full end-to-end KCP
traffic needs an integration client because it depends on UDP sockets and the
nano handshake sequence.
