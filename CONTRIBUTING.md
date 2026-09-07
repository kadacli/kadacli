# Contributing to kåda

Thanks for your interest in improving kåda! This project is in an early phase,
and contributions, bug reports, and ideas are all welcome.

## Before you start

- For anything non-trivial, **please open an issue first** so we can discuss the
  approach before you invest time in a pull request.
- Be kind and constructive. See our [Code of Conduct](CODE_OF_CONDUCT.md).

## Development

kåda is written in Go (1.26+).

```bash
# Fetch dependencies
go mod download

# Build the CLI
go build -o kada .

# Run the tests
go test ./...
```

## Pull requests

1. Fork the repository and create a branch from `main`.
2. Make your change, with tests where it makes sense.
3. Ensure `go build ./...` and `go test ./...` pass.
4. Open a pull request describing **what** changed and **why**.

## Licensing of contributions

kåda is released under the [Apache License 2.0](LICENSE). By submitting a
contribution, you agree that it is licensed under those same terms.

> As the project matures we plan to introduce a Contributor License Agreement
> (CLA) to keep our licensing options open. Until then, all contributions are
> accepted under Apache 2.0.

## Questions

Open a [discussion or issue](https://github.com/kadacli), or email
**hello@kadacli.io**.
