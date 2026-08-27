#!/usr/bin/env python3
import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[2]
RELEASE_PARITY = (ROOT / "scripts" / "run-release-parity-gates.sh").read_text()
DUAL_PARITY = (ROOT / "scripts" / "run-dual-runtime-parity.sh").read_text()
REAL_PARITY = (ROOT / "scripts" / "run-real-parity-smoke.sh").read_text()


class Task26ParityTimeoutContractTest(unittest.TestCase):
    def test_real_runtime_subflows_are_bounded_and_identifiable(self):
        self.assertIn("run_bounded", RELEASE_PARITY)
        self.assertIn("timeout --foreground", RELEASE_PARITY)
        for label in ("G6-G7", "G8", "G11-G13", "G12.1"):
            self.assertIn(f'run_bounded "{label}"', RELEASE_PARITY)
        for script in (
            "run-real-agent-smoke.sh",
            "run-real-api-doc-smoke.sh",
            "run-real-parity-smoke.sh",
            "run-dual-runtime-parity.sh",
        ):
            self.assertIn(script, RELEASE_PARITY)

    def test_dual_runtime_runner_itself_has_a_hard_deadline(self):
        self.assertIn("PARITY_RUNNER_TIMEOUT_SECONDS", DUAL_PARITY)
        self.assertIn('timeout --foreground "$parity_runner_timeout_seconds" go run ./cmd/parity-runner', DUAL_PARITY)

    def test_real_parity_cleanup_cannot_block_forever_after_cancel_streaming(self):
        self.assertIn("terminate_pid", REAL_PARITY)
        self.assertIn('kill -KILL "$pid"', REAL_PARITY)
        self.assertIn("CLEANUP_TERM_GRACE_SECONDS", REAL_PARITY)
        cleanup = REAL_PARITY.split("cleanup() {", 1)[1].split("}\ntrap cleanup", 1)[0]
        self.assertIn('terminate_pid "$opencode_pid"', cleanup)
        self.assertIn('terminate_pid "$fake_pid"', cleanup)
        self.assertNotIn('wait "$opencode_pid"', cleanup)
        self.assertNotIn('wait "$fake_pid"', cleanup)


if __name__ == "__main__":
    unittest.main()
