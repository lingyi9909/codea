import json
import pathlib
import re
import unittest

import yaml

ROOT = pathlib.Path(__file__).resolve().parents[1]


class Task26AcceptanceContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.state = yaml.safe_load((ROOT / "docs/execution-state.yaml").read_text())
        cls.runtime = json.loads((ROOT / "runtime/version.json").read_text())
        cls.release = (ROOT / "packaging/config/release.yaml").read_text()
        cls.windows_install = (ROOT / "packaging/platform/windows/install.ps1").read_text()

    def test_tasks_22_through_25_are_human_accepted_before_task26(self):
        for task_id in ("22", "23", "24", "25"):
            task = self.state["tasks"][task_id]
            self.assertEqual(task["status"], "completed")
            self.assertEqual(task["verificationStatus"], "pass")
            self.assertEqual(task["taskGateStatus"], "pass")
            self.assertIs(task["humanAccepted"], True)

        self.assertEqual(str(self.state["current"]["task"]), "26")
        task26 = self.state["tasks"]["26"]
        self.assertIn(task26["status"], {"in_progress", "awaiting_acceptance"})
        self.assertIs(task26["humanAccepted"], False)

    def test_locked_runtime_and_release_targets_match_v11_contract(self):
        self.assertEqual(self.runtime["openCodeVersion"], "1.18.11")
        self.assertEqual(
            set(self.runtime["platforms"]),
            {"linux-x64", "darwin-arm64", "darwin-x64", "windows-x64"},
        )
        for platform, meta in self.runtime["platforms"].items():
            self.assertIn("v1.18.11", meta["url"], platform)
            self.assertTrue(meta["checksum"].startswith("sha256:"), platform)

        self.assertIn('openCodeVersion: "1.18.11"', self.release)
        for platform in ("darwin-arm64", "darwin-x64", "windows-x64"):
            self.assertIn(platform, self.release)

    def test_windows_release_path_has_no_wsl_runtime_dependency(self):
        self.assertIsNone(
            re.search(
                r"(?i)(^|[^a-z0-9_])wsl(?:\.exe)?([^a-z0-9_]|$)",
                self.windows_install,
            )
        )


if __name__ == "__main__":
    unittest.main()
