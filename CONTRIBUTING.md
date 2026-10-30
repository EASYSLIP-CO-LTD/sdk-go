# Contributing

Describe the customer-visible problem in an issue before changing public behavior. Keep reproductions small and use synthetic slips and credentials.

```sh
go vet ./...
go test -race ./...
staticcheck ./...
golangci-lint run
```

Run these checks on supported Go versions. Add regression tests for behavior changes and update both READMEs when a public option changes. Submit changes through pull requests. Do not include customer data or API keys.

The release workflow prepares version and changelog updates. After approval, the published Git tag is checked through the Go module proxy and an exact-version consumer is built. Nothing is uploaded to a package registry.
