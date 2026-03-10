# Contributing to goqlprinter

## Reporting Bugs

Open a [GitHub Issue](../../issues/new) with:
- Your OS and Go version
- Printer model and connection method (USB/network)
- Steps to reproduce
- Relevant log output

## Proposing Features

Open a GitHub Issue describing the use case before writing code. This avoids effort spent on things that won't be merged.

## Development Setup

Requirements:
- Go 1.24+
- Node.js 22+
- [`just`](https://github.com/casey/just)
- [`mise`](https://mise.jdx.dev/) (optional, manages tool versions)

```bash
git clone https://github.com/your-org/goqlprinter
cd goqlprinter
just install-frontend
```

## Running Locally

```bash
just dev
```

Starts Go backend on `:8000` and Vite dev server on `:5173` concurrently. Open http://localhost:5173 in your browser.

## Building

```bash
just build
```

## Linting

```bash
# Go
golangci-lint run

# Frontend
cd frontend && npx eslint src/
```

## Testing

```bash
go test ./...
```

## Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add support for QL-1115NWB
fix: correct raster padding for 62mm tape
chore: update dependencies
docs: add udev rule example
```

Types: `feat`, `fix`, `chore`, `docs`, `refactor`, `test`, `ci`

## Pull Request Process

1. Fork the repo and create a branch from `master`.
2. Make your changes with tests where applicable.
3. Ensure `go test ./...` and linting pass.
4. Open a PR with a clear description of what and why.
5. PRs are squash-merged; keep the title in Conventional Commits format.

## License

By contributing, you agree your contributions will be licensed under the [MIT License](LICENSE).
