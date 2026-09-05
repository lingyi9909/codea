#!/usr/bin/env python3
import importlib.util
import pathlib
import sys
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[2]
RELEASE_PARITY = (ROOT / "scripts" / "run-release-parity-gates.sh").read_text()
DUAL_PARITY = (ROOT / "scripts" / "run-dual-runtime-parity.sh").read_text()
REAL_PARITY = (ROOT / "scripts" / "run-real-parity-smoke.sh").read_text()
REAL_AGENT_FIXTURE_DIR = ROOT / "tests" / "fixtures" / "real-parity"


def load_real_agent_fake_model():
    module_path = REAL_AGENT_FIXTURE_DIR / "fake_model.py"
    spec = importlib.util.spec_from_file_location("task26_real_agent_fake_model", module_path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"unable to load {module_path}")
    module = importlib.util.module_from_spec(spec)
    sys.path.insert(0, str(REAL_AGENT_FIXTURE_DIR))
    try:
        spec.loader.exec_module(module)
    finally:
        sys.path.pop(0)
    return module


def tool_history(*names):
    messages = []
    for index, name in enumerate(names):
        call_id = f"call_{index}_{name}"
        messages.append(
            {
                "role": "assistant",
                "tool_calls": [
                    {
                        "id": call_id,
                        "type": "function",
                        "function": {"name": name, "arguments": "{}"},
                    }
                ],
            }
        )
        messages.append({"role": "tool", "tool_call_id": call_id, "content": "{}"})
    return messages


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

    def test_real_agent_fixture_plans_before_unit_test_mutations(self):
        fake_model = load_real_agent_fake_model()

        tool, args, _, _ = fake_model.decide("CALL WRITE_TEST_FILE please", [])
        self.assertEqual(tool, "task_plan")
        self.assertEqual(len(args["steps"]), 3)
        tool, _, _, _ = fake_model.decide(
            "CALL WRITE_TEST_FILE please", tool_history("task_plan")
        )
        self.assertEqual(tool, "write_test_file")

        for prompt in ("UTFLOW", "UTFLOW_FAIL"):
            tool, _, _, _ = fake_model.decide(prompt, [])
            self.assertEqual(tool, "analyze_test_project")
            tool, args, _, _ = fake_model.decide(
                prompt, tool_history("analyze_test_project")
            )
            self.assertEqual(tool, "task_plan")
            self.assertEqual(len(args["steps"]), 3)
            tool, _, _, _ = fake_model.decide(
                prompt, tool_history("analyze_test_project", "task_plan")
            )
            self.assertEqual(tool, "write_test_file")
            tool, _, _, _ = fake_model.decide(
                prompt,
                tool_history("analyze_test_project", "task_plan", "write_test_file"),
            )
            self.assertEqual(tool, "run_project_test")


if __name__ == "__main__":
    unittest.main()
