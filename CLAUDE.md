# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**idem** is a Go HTTP middleware library for idempotency key handling. It intercepts requests with an `Idempotency-Key` header, caches responses, and returns cached results for duplicate requests. Designed to be framework-agnostic (net/http compatible with Gin/Echo/Chi) with pluggable storage backends.

Module: `github.com/bright-room/idem` | Go 1.26.1 | Pre-v1.0

## Development Commands

All commands run inside Docker (via `compose.yml`). Use `make build` first to build the image. The `compose.yml` includes standalone Redis, a 3-node Redis Cluster (ports 7000-7002), and a Redis Sentinel setup (master + replica + 3 sentinels on ports 26379-26381) for integration tests.

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

For package structure, key interfaces, and design patterns, see @.claude/rules/architecture.md

## Coding Conventions

For linting, testing, naming, and code generation conventions, see @.claude/rules/coding.md

## CI Pipeline (on-pull-request.yml)

Three parallel jobs: lint, unit test (with octocov coverage), integration test. Test reports via `dorny/test-reporter` in JUnit format.
