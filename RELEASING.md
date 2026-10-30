# Releasing the SDK

## One-time setup

Allow GitHub Actions to create pull requests. Protect `main` and require the normal CI checks before merging. Configure the `release` environment to allow only `main`, add the intended release reviewers and keep integration credentials out of this environment. The release workflow uses the repository's `GITHUB_TOKEN`; it needs no personal access token or GitHub App.

The module path is `github.com/EASYSLIP-CO-LTD/sdk-go`. Make the repository public before releasing so the Go module proxy can fetch its tags. No registry token is needed. Artifact provenance uses OIDC in `.github/workflows/release.yml`, environment `release`.

## Check the version pull request

A push to `main` runs release-please in `.github/workflows/release.yml`. It opens or updates a version and changelog pull request. The first release is `1.0.0`.

Do not depend on an automatic CI run for a pull request written by `GITHUB_TOKEN`. Before merging each updated version pull request:

1. Copy its current head branch and full head commit SHA from GitHub.
2. Open Actions, select CI, choose Run workflow and select that head branch. The equivalent CLI command is `gh workflow run ci.yml --ref <release-pr-head-branch>`.
3. Check that the run's `headSha` is exactly the SHA you copied. Wait for the full runtime matrix, package/consumer and workflow checks to pass. Inspect the separate security checks too.
4. If the pull request changes, dispatch CI again for the new head. Merge only the tested head after the required reviews/checks pass.

CI dispatch executes that branch with read-only repository permissions. It has no publishing credentials and does not use a privileged pull-request event. The optional integration job remains in its separate protected `integration` environment. Dispatching CI never dispatches publication.

## Publish

After the tested version pull request merges, release-please creates the tag and GitHub release. Its `release_created`, `tag_name` and `sha` outputs feed the next jobs in the same workflow run; publication does not wait for another release event.

The target job has read-only permissions. It requires a stable `vMAJOR.MINOR.PATCH` tag, resolves an existing tag to its exact commit, checks that a published GitHub release exists and compares the commit with release-please's output. The publishing job waits for the `release` environment, checks out that exact SHA, and verifies that HEAD, the tag commit and the package version agree. Only this job receives OIDC and attestation permissions.

The job tests and builds the exact release, archives its source and attests the archive. It waits for the exact version at `proxy.golang.org`, installs that version into a fresh consumer module and runs an offline request. Only then does it attach the archive to the GitHub release. A tag is the Go publication; no package upload is needed.

An ordinary push with no new release skips target resolution and publishing. A failed or empty release-please result cannot publish. Runs are serialized and publishing is never cancelled automatically by another push.

## Rerun an existing release

Open Actions, select Release, choose Run workflow on `main`, and enter an existing published tag such as `v1.0.0`. The CLI equivalent is `gh workflow run release.yml --ref main -f tag=v1.0.0`. This path skips release-please entirely: it cannot create a tag or a release. Missing tags, drafts, malformed versions, version mismatches and dispatches from another branch/tag fail closed.

The same `release` environment approval and exact-tag checks apply. Do not move or recreate a released tag. An existing version is immutable; a changed package needs a new version. Manual reruns use the scripts and workflow at the resolved release commit after checkout, so that commit must already contain this release validation.

A proxy delay or failed artifact attachment can be retried without creating a new tag. The exact-version consumer check still runs. Release artifact attachments use the same names and may be replaced after that check succeeds.

## Local validation

Run `python -m unittest discover -s scripts -p test_release.py`, actionlint and zizmor before changing the release chain. Tests cover ordinary pushes, newly created releases, manual selection, invalid/missing tags, changed tag commits and package-version mismatches. They mock GitHub GET responses and never publish.
