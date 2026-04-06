# Contributing to Qon

Thanks for your interest in contributing to Qon.

## Development Setup

1. Install Go 1.22 or newer.
2. Clone the repository.
3. Build and test:

```bash
go build ./...
go test ./...
```

## Branching

1. Create a feature branch from `main`.
2. Keep pull requests focused and small when possible.
3. Rebase on the latest `main` before opening a PR.

## Pull Request Checklist

- Include a clear description of the change and motivation.
- Add or update tests for behavior changes.
- Ensure `go test ./...` passes locally.
- Ensure `go vet ./...` passes locally.
- Update documentation when APIs or behavior change.

## Coding Guidelines

- Prefer explicit behavior over implicit magic.
- Keep allocations and abstractions minimal in hot paths.
- Preserve backward compatibility unless a breaking change is intentional.
- Use structured errors and avoid leaking internal details.

## Commit Messages

Use clear, imperative commit messages, for example:

- `core: add retry jitter support`
- `bridge: validate malformed JSON input`
- `docs: update server mode examples`

## Reporting Bugs and Requesting Features

Use the issue creation on our repository at [qonhq/qon/issues/new](https://github.com/qonhq/qon/issues/new) to report bugs or request features. Please provide as much detail as possible, including steps to reproduce, expected vs actual behavior, and any relevant logs or screenshots.

## Questions

Open a discussion in GitHub Discussions once enabled, or file a question issue.
