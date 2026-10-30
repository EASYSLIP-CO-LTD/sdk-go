# Contributing

Describe the customer-visible problem in an issue before changing public behavior. Keep reproductions small and use synthetic slips and credentials.

```sh
go vet ./...
go test -race ./...
staticcheck ./...
golangci-lint run
```

Run these checks on supported Go versions. Add regression tests for behavior changes and update both READMEs when a public option changes. Submit changes through pull requests. Do not include customer data or API keys.

Release maintainers run CI on the automated version pull request before merging it. `.github/workflows/release.yml` then publishes through the `release` environment in the same run that creates the release. See [the release guide](RELEASING.md) for CI dispatch, approval and existing-tag reruns. No personal access token or GitHub App is required.
