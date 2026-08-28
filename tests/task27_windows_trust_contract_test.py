import pathlib
import unittest

import yaml

ROOT = pathlib.Path(__file__).resolve().parents[1]


class Task27WindowsTrustContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.state = yaml.safe_load((ROOT / "docs/execution-state.yaml").read_text())
        cls.process_windows = (ROOT / "tui/internal/supervisor/process_windows.go").read_text()
        cls.installer = (ROOT / "packaging/platform/windows/install.ps1").read_text()
        cls.workflow = (ROOT / ".github/workflows/task27-windows-trust-gates.yml").read_text()
        cls.lifecycle = (ROOT / "tests/release/task27-windows-installed-lifecycle.ps1").read_text()

    def test_task26_is_accepted_and_task27_is_active(self):
        task26 = self.state["tasks"]["26"]
        self.assertEqual(task26["status"], "completed")
        self.assertIs(task26["humanAccepted"], True)
        self.assertEqual(str(self.state["current"]["task"]), "27")
        self.assertEqual(str(self.state["taskOrder"][-1]), "27")
        task27 = self.state["tasks"]["27"]
        self.assertIn(task27["status"], {"in_progress", "awaiting_acceptance"})
        self.assertIs(task27["humanAccepted"], False)

    def test_windows_start_reliability_layers_are_retained(self):
        self.assertIn("func prepareRuntimeBinary", self.process_windows)
        self.assertIn("syscall.ERROR_ACCESS_DENIED", self.process_windows)
        self.assertIn("runtimeStartMaxAttempts", self.process_windows)
        self.assertIn("runtimeStartRetryDelay", self.process_windows)
        self.assertIn("Get-FileHash -Algorithm SHA256", self.installer)
        self.assertIn("size mismatch", self.installer)
        self.assertIn("Unblock-File", self.installer)

    def test_job_object_uses_minimal_process_rights(self):
        self.assertNotIn("processAllAccess", self.process_windows)
        self.assertIn("processJobAccess", self.process_windows)
        self.assertIn("processSetQuota", self.process_windows)
        self.assertIn("processTerminate", self.process_windows)

    def test_real_windows_lifecycle_and_signing_are_wired(self):
        required = [
            "task27-windows-installed-lifecycle.ps1",
            "sign-release.ps1",
            "verify-signature.ps1",
            "Full native Windows Go regression",
            "TestStopTerminatesProcessTree",
            "TestStartRuntimeCommandRetriesTransientAccessDenied",
            "fresh install",
            "MOTW",
            "upgrade",
            "rollback",
        ]
        for token in required:
            self.assertIn(token, self.workflow)

    def test_lifecycle_diagnostics_are_powershell_parse_safe(self):
        self.assertIn("${Scenario}:", self.lifecycle)
        self.assertNotIn("$Scenario:", self.lifecycle)


if __name__ == "__main__":
    unittest.main()
