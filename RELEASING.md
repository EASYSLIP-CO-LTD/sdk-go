# Releasing

## One-time setup

- npm: configure trusted publishing for `release.yml`, environment `release`. Publish the first npm version by hand.
- PyPI: configure a trusted publisher for `release.yml`, environment `release`.
- Packagist: submit the package and connect its GitHub hook.
- Go: no setup is needed.

## Release

Merge the release pull request.

## Re-run

Re-run the failed job, or run Release on the version tag.
