# Repository Guidelines

## Project Structure & Module Organization

Nano is a Go module (`github.com/lonng/nano`) for game server networking. Core files live at the repository root, with packages grouped by responsibility:

- `component/`, `service/`, `session/`, `pipeline/`, and `scheduler/` contain the primary runtime APIs.
- `cluster/` contains distributed server support and generated protobuf code under `cluster/clusterpb/`.
- `internal/` holds private codec, packet, message, runtime, environment, and logging helpers.
- `serialize/` contains JSON and protobuf serializers.
- `examples/` contains runnable demos and client assets.
- `docs/`, `media/`, and `LEARNING_GUIDE_zh_CN.md` contain documentation and diagrams.
- Tests are colocated as `*_test.go`; integration scripts live in `tests/`; benchmarks live in `benchmark/`.

## Build, Test, and Development Commands

- `go test -v ./...` or `make test`: run all unit tests across the module.
- `go test -v ./cluster ./service`: run focused package tests while iterating.
- `go test -v -tags "benchmark"` from `benchmark/io`: run benchmark-tagged IO tests.
- `make proto`: regenerate cluster protobuf Go files from `cluster/clusterpb/proto/*.proto`.
- `go run ./examples/demo/chat`: run the chat demo server locally.

Install `protoc`, `protoc-gen-go`, and gRPC Go plugins before changing protobuf definitions.

## Coding Style & Naming Conventions

Use standard Go formatting: run `gofmt` on changed Go files before committing. Follow Go community style from CodeReviewComments. Package names are short, lowercase, and descriptive (`session`, `serialize`, `scheduler`). Exported identifiers should use clear names and comments where needed. Keep generated files, such as `*.pb.go`, updated from source `.proto` files.

## Testing Guidelines

Add or update colocated `*_test.go` files for bug fixes and new behavior. Name tests `TestXxx` and use table-driven cases for multiple inputs. Prefer package-level tests for public behavior and internal package tests for codec, packet, and message details. Run `go test -v ./...` before submitting.

## Commit & Pull Request Guidelines

Follow the repository convention:

```text
<subsystem>: <what changed>

<why this change was made>
```

Keep the subject under 70 characters and wrap body lines near 80 characters. Use subsystem names such as `cluster:`, `session:`, `serialize:`, or `*:` for broad changes. Pull requests should describe the change, explain why it is needed, mention tests run, and link related issues. Include screenshots only when changing web demo assets.

## Agent-Specific Instructions

Act as a senior game server architect. Prefer architectures that have been proven in production at large-scale studios or platform teams: clear service boundaries, deterministic state ownership, explicit message flow, backpressure, observability, and failure isolation. If a proposed design is risky, overcomplicated, or mismatched to this codebase, challenge it directly and explain the trade-off. Always offer a simpler, more maintainable alternative with concrete implementation steps.

Keep edits focused and consistent with existing package boundaries. Do not rewrite generated protobuf files by hand. Avoid unrelated formatting churn, especially in examples and documentation.
