# Contributing to gokalshi

Thank you for your interest in contributing to gokalshi!

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/<your-username>/gokalshi.git`
3. Create a feature branch: `git checkout -b feat/my-change`
4. Make your changes
5. Push and open a pull request

## Development

### Prerequisites

- Go 1.26+
- [golangci-lint](https://golangci-lint.run/welcome/install/)

### Running Tests

```bash
# Unit tests
go test ./... -v

# Unit tests with coverage
go test ./... -v -coverprofile=coverage.out
go tool cover -func=coverage.out

# Unit tests with race detector
go test -race ./... -count=1

# Integration tests (requires Kalshi API credentials)
go test -tags=integration ./... -v

# Spec validation (fetches live OpenAPI/AsyncAPI specs)
go test -tags=spec_validation ./... -v
```

### Linting

```bash
golangci-lint run
```

## Pull Request Guidelines

- **PR titles** must follow [Conventional Commits](https://www.conventionalcommits.org/) format:
  `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `ci:`, `chore:`, `perf:`, `style:`
- Keep changes focused — one concern per PR
- Add tests for new functionality
- Ensure all existing tests pass
- Run `golangci-lint run` before submitting

## Design Principles

This SDK is a **pure 1:1 reflection** of the Kalshi API:

- Each method maps to exactly one API endpoint
- No pagination helpers, no delta polling, no caching
- Application-level abstractions belong in the consumer, not here
- Generated types come from the OpenAPI spec via `go generate`

## Reporting Issues

Use [GitHub Issues](https://github.com/Kalarb/gokalshi/issues) to report bugs or request features. For security vulnerabilities, see [SECURITY.md](SECURITY.md).

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
