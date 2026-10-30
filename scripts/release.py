"""Resolve existing release tags and verify checked-out package versions."""

import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tomllib
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

TAG = re.compile(r"v(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)")
SHA = re.compile(r"[0-9a-f]{40}")


def selected_target(env: dict[str, str]) -> tuple[str, str] | None:
    if env.get("GITHUB_REF") != "refs/heads/main":
        raise ValueError("Release workflow must run from main")
    event = env.get("GITHUB_EVENT_NAME")
    if event == "push":
        if env.get("RELEASE_CREATED") != "true":
            return None
        tag, sha = env.get("AUTOMATIC_TAG", ""), env.get("AUTOMATIC_SHA", "")
        if not SHA.fullmatch(sha):
            raise ValueError("Release output has no valid commit")
    elif event == "workflow_dispatch":
        tag, sha = env.get("INPUT_TAG", ""), ""
    else:
        raise ValueError("Unsupported release event")
    if not TAG.fullmatch(tag):
        raise ValueError("Expected a stable v-prefixed version tag")
    return tag, sha


def api(path: str) -> dict[str, object]:
    request = Request(
        "https://api.github.com/repos/" + os.environ["GITHUB_REPOSITORY"] + "/" + path,
        headers={
            "Authorization": "Bearer " + os.environ["GH_TOKEN"],
            "Accept": "application/vnd.github+json",
            "User-Agent": "sdk-release-validation",
            "X-GitHub-Api-Version": "2022-11-28",
        },
    )
    try:
        with urlopen(request, timeout=20) as response:
            result = json.load(response)
    except (HTTPError, URLError):
        raise ValueError("Release lookup failed; check tag, release and read permissions") from None
    if not isinstance(result, dict):
        raise ValueError("Unexpected release lookup response")
    return result


def resolve_target(tag: str, expected: str, lookup=api) -> str:
    if not TAG.fullmatch(tag):
        raise ValueError("Invalid release tag")
    ref = lookup("git/ref/tags/" + tag)
    if ref.get("ref") != "refs/tags/" + tag:
        raise ValueError("Tag does not exist")
    commit = lookup("commits/" + tag)
    sha = commit.get("sha", "")
    if not isinstance(sha, str) or not SHA.fullmatch(sha):
        raise ValueError("Invalid tag commit")
    if expected and expected != sha:
        raise ValueError("Tag commit differs from release output")
    release = lookup("releases/tags/" + tag)
    if release.get("tag_name") != tag or release.get("draft") is not False:
        raise ValueError("Expected an existing published GitHub release")
    return sha


def validate_checkout(root: Path, tag: str, sha: str) -> None:
    if not TAG.fullmatch(tag) or not SHA.fullmatch(sha):
        raise ValueError("Invalid release identity")
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
    tagged = subprocess.check_output(
        ["git", "rev-parse", "refs/tags/" + tag + "^{commit}"], cwd=root, text=True
    ).strip()
    if head != sha or tagged != sha:
        raise ValueError("Checkout does not match the resolved tag commit")
    if (root / "pyproject.toml").exists():
        with (root / "pyproject.toml").open("rb") as file:
            version = tomllib.load(file)["project"]["version"]
    else:
        versions = re.findall(
            r'agent \+ "/([0-9]+\.[0-9]+\.[0-9]+) go/".*// x-release-please-version',
            (root / "client.go").read_text(),
        )
        if len(versions) != 1:
            raise ValueError("Package version marker is missing or ambiguous")
        version = versions[0]
    if tag != "v" + version:
        raise ValueError("Tag does not equal the package version")


def main() -> None:
    if sys.argv[1:] == ["resolve"]:
        target = selected_target(dict(os.environ))
        if target is None:
            raise ValueError("No new release to publish")
        tag, expected = target
        sha = resolve_target(tag, expected)
        with open(os.environ["GITHUB_OUTPUT"], "a") as output:
            output.write(f"tag={tag}\nsha={sha}\n")
    elif sys.argv[1:] == ["check"]:
        validate_checkout(Path.cwd(), os.environ["RELEASE_TAG"], os.environ["RELEASE_SHA"])
    else:
        raise ValueError("Use resolve or check")


if __name__ == "__main__":
    try:
        main()
    except ValueError as error:
        sys.exit(str(error))
