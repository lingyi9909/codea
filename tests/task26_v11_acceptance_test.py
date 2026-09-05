import importlib.util
import json
import pathlib
import re
import unittest

import yaml

ROOT = pathlib.Path(__file__).resolve().parents[1]


def load_fake_api_doc_model():
    module_path = ROOT / "tests/fixtures/real-parity/fake_api_doc_model.py"
    spec = importlib.util.spec_from_file_location("fake_api_doc_model_contract", module_path)
    if spec is None or spec.loader is None:
        raise RuntimeError("cannot load fake_api_doc_model.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def load_release_parity_model():
    module_path = ROOT / "tests/fixtures/release-parity/fake_model.py"
    spec = importlib.util.spec_from_file_location("release_parity_fake_model_contract", module_path)
    if spec is None or spec.loader is None:
        raise RuntimeError("cannot load release-parity fake_model.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def assistant_tool(name, call_id):
    return {
        "role": "assistant",
        "tool_calls": [{"id": call_id, "type": "function", "function": {"name": name, "arguments": "{}"}}],
    }


def advertised_tools(*names):
    return [
        {
            "type": "function",
            "function": {
                "name": name,
                "description": f"contract tool {name}",
                "parameters": {"type": "object", "properties": {}},
            },
        }
        for name in names
    ]


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

        order = [str(task_id) for task_id in self.state["taskOrder"]]
        current_task = str(self.state["current"]["task"])
        self.assertIn("26", order)
        self.assertIn(current_task, order)
        self.assertLessEqual(order.index("26"), order.index(current_task))

        task26 = self.state["tasks"]["26"]
        self.assertEqual(task26["verificationStatus"], "pass")
        self.assertEqual(task26["taskGateStatus"], "pass")

        if current_task == "26":
            if task26["status"] == "completed":
                self.assertIs(task26["humanAccepted"], True)
                self.assertEqual(self.state["current"]["status"], "completed")
                self.assertIs(self.state["humanAcceptance"]["accepted"], True)
            else:
                self.assertIn(task26["status"], {"in_progress", "awaiting_acceptance"})
                self.assertIs(task26["humanAccepted"], False)
                self.assertIs(self.state["humanAcceptance"]["accepted"], False)
        else:
            self.assertEqual(task26["status"], "completed")
            self.assertIs(task26["humanAccepted"], True)

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

    def test_api_doc_real_parity_fixture_plans_before_document_mutation(self):
        model = load_fake_api_doc_model()

        single_messages = [{"role": "user", "content": "CALL WRITE_DOCUMENT please"}]
        name, args, call_id, _ = model.decide("CALL WRITE_DOCUMENT please", single_messages)
        self.assertEqual(name, "task_plan")
        self.assertGreaterEqual(len(args["steps"]), 3)
        single_messages.append(assistant_tool("task_plan", call_id))
        name, _, _, _ = model.decide("CALL WRITE_DOCUMENT please", single_messages)
        self.assertEqual(name, "write_document")

        spec = {
            "controllerName": "DemoController",
            "basePath": "/api",
            "endpoints": [{"method": "POST", "path": "/users", "requestBody": {"type": "CreateUserRequest"}, "errorCodes": []}],
            "dtos": {},
            "enums": {},
        }
        flow_messages = [
            {"role": "user", "content": "APIDOCFLOW generate the API documentation"},
            assistant_tool("extract_api_spec", "call_api_extract"),
            {"role": "tool", "tool_call_id": "call_api_extract", "content": json.dumps(spec)},
            assistant_tool("validate_api_example", "call_api_validate"),
            {"role": "tool", "tool_call_id": "call_api_validate", "content": json.dumps({"valid": True, "errors": [], "warnings": []})},
        ]
        name, args, call_id, _ = model.decide("APIDOCFLOW generate the API documentation", flow_messages)
        self.assertEqual(name, "task_plan")
        self.assertGreaterEqual(len(args["steps"]), 3)
        flow_messages.append(assistant_tool("task_plan", call_id))
        name, _, _, _ = model.decide("APIDOCFLOW generate the API documentation", flow_messages)
        self.assertEqual(name, "write_document")

    def test_release_parity_fixture_plans_candidate_approval_before_bash(self):
        model = load_release_parity_model()
        baseline_tools = advertised_tools("bash")
        candidate_tools = advertised_tools("task_plan", "task_step", "task_status", "bash")

        for prompt in ("APPROVAL TEST", "REJECT TEST"):
            with self.subTest(prompt=prompt, runtime="baseline"):
                name, _, _, _ = model.decide(prompt, [{"role": "user", "content": prompt}], baseline_tools)
                self.assertEqual(name, "bash")

            with self.subTest(prompt=prompt, runtime="candidate"):
                messages = [{"role": "user", "content": prompt}]
                name, args, call_id, _ = model.decide(prompt, messages, candidate_tools)
                self.assertEqual(name, "task_plan")
                self.assertGreaterEqual(len(args["steps"]), 3)
                messages.append(assistant_tool("task_plan", call_id))
                messages.append({"role": "tool", "tool_call_id": call_id, "content": '{"status":"active"}'})
                name, _, _, _ = model.decide(prompt, messages, candidate_tools)
                self.assertEqual(name, "bash")


if __name__ == "__main__":
    unittest.main()
