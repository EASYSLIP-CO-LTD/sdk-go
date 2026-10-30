# Contributing

Discuss bugs and public API changes in an issue before starting a large change. For vulnerabilities, follow [SECURITY.md](SECURITY.md). Use synthetic data; never include API keys, webhook secrets or real slip images.

## Setup and checks

Clone this repository, enter its directory, then run the commands below. Use a runtime supported by the CI matrix in `.github/workflows/ci.yml`.

```sh
go mod download
go vet ./...
go test -race ./...
staticcheck ./...
golangci-lint run
go run ./scripts/check-readme
go build ./...
```

These checks run locally without an API key. Live integration tests are optional, use your own credentials and may use quota. Never commit their credentials or slip data.

## Pull request checklist

- Keep the change focused and explain the customer-visible behavior.
- Add regression tests for fixes and tests for new behavior.
- Update both `README.md` and `README.th.md` when the public API changes.
- Run the checks above and include their results in the pull request.
- Check that the diff contains no credentials or customer data.
- Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for commits and the pull request title, such as `fix: preserve request ids`. Use `feat` for a feature, `fix` for a fix and `!` for a breaking change.

Maintainers follow [RELEASING.md](RELEASING.md). Publishing runs in `.github/workflows/release.yml`, environment `release`.
