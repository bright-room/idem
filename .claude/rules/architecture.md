---
paths:
  - "**/*.go"
---

# Architecture

## Package Structure

- **Root package (`idem`)**: Core types and middleware -- `idem.go`, `middleware.go`, `storage.go`, `locker.go`, `response.go`, `option.go`. Generated files: `recorder_gen.go`, `recorder_gen_test.go`
- **`internal/cmd/genrecorder/`**: Code generator for `responseRecorder` combination types (run via `go generate ./...`)
- **`gin/`**: Gin framework adapter (separate sub-module with own `go.mod` to isolate the `github.com/gin-gonic/gin` dependency)
- **`redis/`**: Redis storage implementation
- **`_examples/`**: Framework-specific usage examples (Gin, Echo, Chi)

## Key Interfaces

- `Storage` -- `Get`/`Set`/`Delete` for cached responses (pluggable backend)
- `Locker` -- optional per-key mutual exclusion for concurrent request handling. Storage implementations that also implement `Locker` enable automatic lock acquisition in the middleware. Returns 409 Conflict on lock failure.
- `Validator` -- `Validate(Config) error` for configuration validation. `ValidatorFunc` is a function adapter (like `http.HandlerFunc`). Preset validators (`MaxTTL`, `MinTTL`, etc.) return `*PresetValidator` which supports `.WithMessage()` for custom error messages. Composition functions `All` (AND) and `Any` (OR) combine multiple validators and also return `*PresetValidator`.

## Key Types

- `CacheableFunc` -- `func(statusCode int) bool` that determines whether a response should be cached. `DefaultCacheable` caches 1xx–4xx and skips 5xx. Configurable via `WithCacheable`.
- `Duration` -- `time.Duration` wrapper with human-readable JSON serialization (e.g. `"1h0m0s"` instead of nanoseconds). Used in `Config.TTL`.
- `ConfigDiff` / `FieldDiff` -- `DiffConfig(a, b Config) ConfigDiff` compares two `Config` snapshots and returns structured field-level differences. `HasDiff()` checks for any change; `String()` renders a human-readable summary.

## Design Patterns

- The `responseRecorder` implements `Unwrap() http.ResponseWriter`, enabling `http.ResponseController` to traverse the wrapper chain and discover interfaces (e.g. `SetReadDeadline`, `SetWriteDeadline`) on the underlying writer.
- Configuration uses the **Functional Options** pattern (`WithXxx()` functions).
