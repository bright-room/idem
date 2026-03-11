# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**idem** is a Go HTTP middleware library for idempotency key handling. It intercepts requests with an `Idempotency-Key` header, caches responses, and returns cached results for duplicate requests. Designed to be framework-agnostic (net/http compatible with Gin/Echo/Chi) with pluggable storage backends.

Module: `github.com/bright-room/idem` | Go 1.25.5 | Pre-v1.0

## Development Commands

All commands run inside Docker (via `compose.yml`). Use `make build` first to build the image.

```bash
make build          # Build Docker image
make fmt            # Format code (golangci-lint / gofumpt)
make lint           # Run golangci-lint
make test           # All tests with coverage and race detection
make test-unit      # Short tests only (-short flag)
make test-integration  # Integration tests (-run Integration -count=1)
make godoc          # Start godoc server on :6060
make shell          # Interactive bash in container
```

## Architecture

- **Root package (`idem`)**: Core types and middleware — `idem.go`, `middleware.go`, `storage.go`, `response.go`, `option.go`
- **`memory/`**: In-memory storage implementation
- **`redis/`**: Redis storage implementation
- **`_examples/`**: Framework-specific usage examples (planned v0.3)

Key interfaces:
- `Storage` — `Get`/`Set` for cached responses (pluggable backend)
- `Locker` — distributed locking for concurrent request handling (planned v0.4)

Configuration uses the **Functional Options** pattern (`WithXxx()` functions).

## Conventions

- **Linting**: golangci-lint v2.10.1 with gofumpt formatting. Zero tolerance (no issue limits). See `.golangci.yml` for enabled linters.
- **Testing**: Table-driven tests, `gotestsum` with testdox output, `-race` flag. Integration tests use `Integration` prefix in test names.
- **Naming**: Short receiver names (`s`, `m`). Interface names use "-er" suffix.
- **Git branching**: `feat/issue-number-description` or `fix/issue-number-description`. PR titles prefixed with `Close #IssueNumber`.

## CI Pipeline (on-pull-request.yml)

Three parallel jobs: lint, unit test (with octocov coverage), integration test. Test reports via `dorny/test-reporter` in JUnit format.

## Custom Skills

- `/plan` — Generate implementation plan from a GitHub Issue
- `/implement` — Execute an implementation plan markdown (creates branch, implements, runs checks, opens PR)
