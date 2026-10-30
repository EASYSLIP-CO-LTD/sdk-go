# Releasing the SDK

## One-time setup

Protect `main` and require CI checks and reviews. Allow GitHub Actions to create pull requests in the repository's Actions settings. Create the `release` environment, limit it to `main` and add release reviewers if approval is required. Keep live integration credentials in the separate `integration` environment. Enable private vulnerability reporting.

All release work runs in `.github/workflows/release.yml`. It uses `GITHUB_TOKEN`, with no personal access token or GitHub App. Publishing and provenance permissions belong only to the jobs that need them.

| SDK | Registry setup |
| --- | --- |
| JavaScript | npm package `@easyslip/sdk`, repository `EASYSLIP-CO-LTD/sdk-js`. Create the `easyslip` npm organization and require two-factor authentication. After the first manual upload, configure trusted publishing for workflow `release.yml`, environment `release`. |
| PHP | Submit `https://github.com/EASYSLIP-CO-LTD/sdk-php` to Packagist as `easyslip/sdk`. Connect its GitHub hook so tags update the package. No Packagist token is stored in CI. |
| Python | Configure a PyPI trusted publisher: owner `EASYSLIP-CO-LTD`, repository `sdk-python`, workflow `release.yml`, environment `release`. These fields must match exactly. No PyPI API token is used. |
| Go | Module `github.com/EASYSLIP-CO-LTD/sdk-go`. Make the repository public before releasing so `proxy.golang.org` can fetch the version tag. No registry token is needed. |

## Check the release pull request

A push to `main` runs release-please and updates a version and changelog pull request. The first release is `1.0.0`. GitHub does not automatically run PR workflows for changes made by `GITHUB_TOKEN`.

The JavaScript and PHP release jobs dispatch CI on the release branch. If checks are missing, or for Python and Go, dispatch CI yourself:

1. Copy the release pull request's current head branch and full commit SHA.
2. Run `gh workflow run ci.yml --ref <release-pr-head-branch>` or select that branch under Actions, CI, Run workflow.
3. Confirm the run's `headSha` matches the copied SHA. Wait for the full runtime matrix, package/consumer checks and security checks.
4. If the pull request changes, run CI again for its new head. Merge only the tested head after reviews and required checks pass.

CI dispatch has read-only repository permissions, never uses `pull_request_target` and does not publish packages. Optional integration tests remain behind their separate environment approval.

## First release and later releases

Merge the tested version pull request. The same workflow run uses release-please's `release_created`, `tag_name` and `sha` outputs to select publication. An ordinary push without a new release cannot publish. Runs are serialized; another push does not cancel a publish in progress.

Before publication, the workflow resolves an existing stable `vMAJOR.MINOR.PATCH` tag, checks its exact commit and validates the package version. JavaScript and PHP require that commit on `main`, build an archive and create provenance, then publish the draft GitHub Release after registry checks. Python and Go require an existing published GitHub release, check out the exact SHA and validate HEAD, tag and package version before publication.

| SDK | Publication and smoke check |
| --- | --- |
| JavaScript | Publish the tarball with npm trusted publishing and provenance, attach it and the attestation, publish the draft release, then install the exact version on supported Node.js versions. |
| PHP | Wait for Packagist to list the version with the tagged source SHA, attach the archive and attestation, publish the draft release, then install the exact version on supported PHP versions. |
| Python | Build and check the wheel and source distribution, publish with PyPI trusted publishing and attestations, install the exact version in a fresh consumer, then attach artifacts to the release. |
| Go | Test and archive the exact source, attest it, wait for the exact version on the Go proxy, build and run a fresh consumer, then attach the archive. The tag is the publication. |

### JavaScript first upload

npm accepts a trusted publisher only after the package exists. The first run therefore stops at Publish if `@easyslip/sdk` does not exist:

1. Download and unpack that run's `package` artifact. Verify it with `gh attestation verify easyslip-sdk-1.0.0.tgz --repo EASYSLIP-CO-LTD/sdk-js`.
2. Using your own npm login, run `npm publish easyslip-sdk-1.0.0.tgz --access public --provenance=false`. This first version has the workflow's build attestation but no npm provenance.
3. Configure the trusted publisher in npm package settings, or use npm 11.15 or later: `npm trust github @easyslip/sdk --repository EASYSLIP-CO-LTD/sdk-js --file release.yml --environment release --allow-publish`.
4. Re-run the failed jobs. The workflow accepts the existing version only when its integrity matches the exact tarball, then completes release attachment and smoke checks. Later releases use trusted publishing with provenance.

A missing npm version is accepted only for the structured `E404` response. Authentication, network and malformed-response errors stop publication.

## Rerun an existing release

Fix the cause and re-run the failed jobs while the original run is available, reusing its artifacts. Otherwise run `gh workflow run release.yml --ref main -f tag=v1.0.0` for an existing version tag. Manual dispatch skips release-please and cannot create a new tag. The same `release` environment and tag/version checks apply. Python and Go also require that the GitHub release is already published; drafts are rejected.

Never move a released tag or replace a published package. A changed package needs a new version. JavaScript accepts an already-published version only with matching tarball integrity. PyPI skips existing files and still runs the exact-version consumer check. PHP checks Packagist's version and source SHA. Go checks the exact proxy version. A rerun can finish a partial release but cannot repair an incorrect published version. Published JavaScript/PHP release assets stay unchanged; Python/Go artifact attachments may be replaced after successful consumer checks.

Run actionlint, zizmor and the repository's release regression tests before changing this workflow. Local tests must mock publishing and never upload a package or create a tag.
