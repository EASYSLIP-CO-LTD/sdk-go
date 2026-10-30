"""Check release event gating, tag resolution and checked-out package identity."""

from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import release

A = "a" * 40
B = "b" * 40


class ReleaseTests(unittest.TestCase):
    def test_ordinary_push_never_publishes(self):
        for value in ("", "false", "True", "1"):
            with self.subTest(value=value):
                self.assertIsNone(
                    release.selected_target(
                        {
                            "GITHUB_EVENT_NAME": "push",
                            "GITHUB_REF": "refs/heads/main",
                            "RELEASE_CREATED": value,
                        }
                    )
                )

    def test_new_release_uses_all_action_outputs(self):
        self.assertEqual(
            release.selected_target(
                {
                    "GITHUB_EVENT_NAME": "push",
                    "GITHUB_REF": "refs/heads/main",
                    "RELEASE_CREATED": "true",
                    "AUTOMATIC_TAG": "v1.0.0",
                    "AUTOMATIC_SHA": A,
                }
            ),
            ("v1.0.0", A),
        )

    def test_manual_rerun_selects_tag_without_creating_release(self):
        self.assertEqual(
            release.selected_target(
                {
                    "GITHUB_EVENT_NAME": "workflow_dispatch",
                    "GITHUB_REF": "refs/heads/main",
                    "INPUT_TAG": "v1.0.0",
                }
            ),
            ("v1.0.0", ""),
        )

    def test_untrusted_refs_and_other_events_are_denied(self):
        for event, ref in (
            ("workflow_dispatch", "refs/heads/feature"),
            ("workflow_dispatch", "refs/tags/v1.0.0"),
            ("release", "refs/heads/main"),
            ("pull_request", "refs/heads/main"),
        ):
            with self.subTest(event=event, ref=ref), self.assertRaises(ValueError):
                release.selected_target({"GITHUB_EVENT_NAME": event, "GITHUB_REF": ref})

    def test_invalid_tags_and_missing_commit_are_denied(self):
        for tag in ("", "1.0.0", "v01.0.0", "v1.0.0-rc.1", "v1.0.0/next", "v1.0.0\n", "$(false)"):
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                release.selected_target(
                    {
                        "GITHUB_EVENT_NAME": "workflow_dispatch",
                        "GITHUB_REF": "refs/heads/main",
                        "INPUT_TAG": tag,
                    }
                )
        with self.assertRaises(ValueError):
            release.selected_target(
                {
                    "GITHUB_EVENT_NAME": "push",
                    "GITHUB_REF": "refs/heads/main",
                    "RELEASE_CREATED": "true",
                    "AUTOMATIC_TAG": "v1.0.0",
                }
            )

    @staticmethod
    def lookup(path):
        return {
            "git/ref/tags/v1.0.0": {"ref": "refs/tags/v1.0.0"},
            "commits/v1.0.0": {"sha": A},
            "releases/tags/v1.0.0": {"tag_name": "v1.0.0", "draft": False},
        }[path]

    def test_lightweight_or_annotated_tag_resolves_to_commit(self):
        self.assertEqual(release.resolve_target("v1.0.0", A, self.lookup), A)
        self.assertEqual(release.resolve_target("v1.0.0", "", self.lookup), A)

    def test_missing_tag_draft_and_changed_tag_are_denied(self):
        for bad_path, bad_result in (
            ("git/ref/tags/v1.0.0", {}),
            ("commits/v1.0.0", {"sha": "invalid"}),
            ("releases/tags/v1.0.0", {"tag_name": "v1.0.0", "draft": True}),
        ):

            def lookup(path):
                return bad_result if path == bad_path else self.lookup(path)

            with self.subTest(path=bad_path), self.assertRaises(ValueError):
                release.resolve_target("v1.0.0", A, lookup)
        with self.assertRaises(ValueError):
            release.resolve_target("v1.0.0", B, self.lookup)

    def test_checkout_tag_and_package_version_must_all_agree(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for filename, contents in (
                ("pyproject.toml", '[project]\nversion = "1.0.0"\n'),
                ("client.go", 'agent + "/1.0.0 go/" // x-release-please-version\n'),
            ):
                with self.subTest(filename=filename):
                    path = root / filename
                    path.write_text(contents)
                    with patch("release.subprocess.check_output", side_effect=[A + "\n", A + "\n"]):
                        release.validate_checkout(root, "v1.0.0", A)
                    with (
                        patch("release.subprocess.check_output", side_effect=[A, B]),
                        self.assertRaises(ValueError),
                    ):
                        release.validate_checkout(root, "v1.0.0", A)
                    with (
                        patch("release.subprocess.check_output", side_effect=[B, A]),
                        self.assertRaises(ValueError),
                    ):
                        release.validate_checkout(root, "v1.0.0", A)
                    path.write_text(contents.replace("1.0.0", "2.0.0"))
                    with (
                        patch("release.subprocess.check_output", side_effect=[A, A]),
                        self.assertRaises(ValueError),
                    ):
                        release.validate_checkout(root, "v1.0.0", A)
                    path.unlink()

    def test_workflow_chain_and_trust_boundaries(self):
        root = Path(__file__).resolve().parents[1]
        workflow = (root / ".github/workflows/release.yml").read_text()
        self.assertIn("needs: [release-please, target]", workflow)
        self.assertIn("needs.release-please.outputs.release_created == 'true'", workflow)
        self.assertIn("ref: ${{ needs.target.outputs.sha }}", workflow)
        self.assertIn("environment: release", workflow)
        self.assertEqual(workflow.count("id-token: write"), 1)
        self.assertEqual(workflow.count("attestations: write"), 1)
        self.assertNotIn("types: [published]", workflow)
        self.assertFalse((root / ".github/workflows/publish.yml").exists())
        self.assertIn("workflow_dispatch:", (root / ".github/workflows/ci.yml").read_text())
        self.assertNotIn("pull_request_target", (root / ".github/workflows/ci.yml").read_text())


if __name__ == "__main__":
    unittest.main()
