#!/usr/bin/env python3
"""Deterministic OpenAI-compatible streaming model stub for the real-parity smoke.

This is NOT a real LLM. It scripts a fixed tool-call lifecycle so the Codea
real-runtime smoke can exercise OpenCode's native Read/Write/Edit/Bash/Task
(subagent)/Skill tools AND the enterprise Custom Tools end-to-end without a
network or an API key. The OpenCode runtime, Agent Loop, message persistence,
permission gating and SSE are all real; only the model is a stub (same
methodology as the S2/S3 Phase 0 spikes).

State machine, keyed on the last user message (all upper-cased):

  Native single-shot tools (emit once, then answer text):
    READ      -> `read` on SMOKE_DIR/read-me.txt
    WRITE     -> `write` creating SMOKE_DIR/write-out.txt
    EDIT      -> `edit` on SMOKE_DIR/edit-me.txt
    BASH      -> `bash` (echo smoke-bash-ok)
    SUBAGENT  -> `task` delegating to the `explore` subagent
    SKILL     -> `skill` for `smoke-skill`

  Enterprise Custom Tool single-shot (whitelist proof):
    COLLECT_REVIEW_CONTEXT -> `collect_review_context` (source=staged)
    WRITE_TEST_FILE        -> `write_test_file` (a fresh whitelist-only test)
    RUN_PROJECT_TEST       -> `run_project_test` (maven)
    WRITE_DOCUMENT         -> `write_document` (docs/smoke.md)

  Enterprise workflow (multi-step, tracked via the assistant tool-call history):
    REVIEWFLOW  -> collect_review_context -> final output-schema JSON answer
    UTFLOW      -> analyze_test_project -> write_test_file -> run_project_test
                   -> final PASS/FAIL derived from the run_project_test Tool Result
    UTFLOW_FAIL -> same chain with a deterministic failing JUnit -> final FAIL
                   derived from the run_project_test Tool Result

A request whose final message is a `tool` result answers with text unless a
workflow still has a remaining step. Keyword matching is ordered so the more
specific custom-tool keywords (which contain WRITE/RUN substrings) win over the
shorter native WRITE, and REVIEWFLOW/UTFLOW win over single-shot tool names.
"""
import json
import os
import sys
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

SMOKE_DIR = os.environ.get("SMOKE_DIR", "/tmp")

WHITELIST_TEST_PATH = "src/test/java/com/example/demo/WhitelistSmokeTest.java"
FLOW_TEST_PATH = "src/test/java/com/example/demo/GeneratedFlowTest.java"
FLOW_TEST_CLASS = "com.example.demo.GeneratedFlowTest"
FAIL_FLOW_TEST_PATH = "src/test/java/com/example/demo/GeneratedFailureFlowTest.java"
FAIL_FLOW_TEST_CLASS = "com.example.demo.GeneratedFailureFlowTest"

WHITELIST_TEST_CONTENT = (
    "package com.example.demo;\n"
    "import org.junit.jupiter.api.Test;\n"
    "class WhitelistSmokeTest {\n"
    "  @Test void whitelist() {}\n"
    "}\n"
)
FLOW_TEST_CONTENT = (
    "package com.example.demo;\n"
    "import org.junit.jupiter.api.Test;\n"
    "class GeneratedFlowTest {\n"
    "  @Test void flow() {}\n"
    "}\n"
)
FAIL_FLOW_TEST_CONTENT = (
    "package com.example.demo;\n"
    "import org.junit.jupiter.api.Assertions;\n"
    "import org.junit.jupiter.api.Test;\n"
    "class GeneratedFailureFlowTest {\n"
    "  @Test void flow() { Assertions.fail(\"deterministic smoke failure\"); }\n"
    "}\n"
)

REVIEWER_JSON = json.dumps(
    {
        "summary": {"result": "PASS", "text": "reviewed staged change; no blocking findings"},
        "scope": {"type": "staged", "changedFiles": ["read-me.txt"]},
        "findings": [],
        "observations": [],
        "uncertainObservations": [],
        "reviewStats": {
            "filesReviewed": 1,
            "filesChanged": 1,
            "callChainsExpanded": 0,
            "critical": 0,
            "major": 0,
            "minor": 0,
            "suggestions": 0,
        },
        "businessKnowledgeUnavailable": True,
    },
    separators=(",", ":"),
)


def tool_call(tool_name, arguments, call_id):
    return {
        "id": call_id,
        "type": "function",
        "function": {"name": tool_name, "arguments": arguments},
    }


def assistant_tool_names(messages):
    """Return the tool names already emitted by the assistant in this thread."""
    names = []
    for m in messages or []:
        if m.get("role") != "assistant":
            continue
        for tc in m.get("tool_calls") or []:
            fn = tc.get("function") or {}
            name = fn.get("name")
            if name:
                names.append(name)
    return names


def _json_value(value):
    """Best-effort decode of OpenAI tool content without assuming one wrapper."""
    if isinstance(value, (dict, list)):
        return value
    if not isinstance(value, str):
        return None
    text = value.strip()
    if not text:
        return None
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        start = text.find("{")
        end = text.rfind("}")
        if start >= 0 and end > start:
            try:
                return json.loads(text[start : end + 1])
            except json.JSONDecodeError:
                return None
    return None


def _find_test_run_result(value):
    """Find the structured run_project_test result inside any tool wrapper."""
    decoded = _json_value(value)
    if decoded is not None and decoded is not value:
        return _find_test_run_result(decoded)
    if isinstance(value, dict):
        if "category" in value and "exitCode" in value:
            return value
        for child in value.values():
            found = _find_test_run_result(child)
            if found is not None:
                return found
    elif isinstance(value, list):
        for child in value:
            found = _find_test_run_result(child)
            if found is not None:
                return found
    return None


def _tool_result_for_call(messages, call_id):
    for message in reversed(messages or []):
        if message.get("role") != "tool":
            continue
        message_call_id = message.get("tool_call_id")
        if message_call_id and message_call_id != call_id:
            continue
        result = _find_test_run_result(message.get("content"))
        if result is not None:
            return result
    return None


def _int_field(result, name, default):
    value = result.get(name, default)
    if isinstance(value, bool):
        return default
    try:
        return int(value)
    except (TypeError, ValueError):
        return default


def unit_test_conclusion(messages, call_id):
    """Return deterministic JSON whose PASS/FAIL is sourced only from run_project_test."""
    result = _tool_result_for_call(messages, call_id)
    if result is None:
        return json.dumps(
            {
                "result": "FAIL",
                "source": "run_project_test",
                "reason": "missing structured run_project_test result",
            },
            separators=(",", ":"),
        )

    category = str(result.get("category", "")).upper()
    exit_code = _int_field(result, "exitCode", -1)
    passed = _int_field(result, "passed", 0)
    failed = _int_field(result, "failed", 0)
    errors = _int_field(result, "errors", 0)
    final = "PASS" if category == "PASS" and exit_code == 0 and passed >= 1 and failed == 0 and errors == 0 else "FAIL"
    return json.dumps(
        {
            "result": final,
            "source": "run_project_test",
            "category": category,
            "exitCode": exit_code,
            "passed": passed,
            "failed": failed,
            "errors": errors,
        },
        separators=(",", ":"),
    )


def decide(prompt, messages):
    """Return (tool_name, arguments, call_id, final_text)."""
    p = (prompt or "").upper()
    names = assistant_tool_names(messages)
    last_is_tool = bool(messages) and messages[-1].get("role") == "tool"

    # --- Enterprise workflows (multi-step) ---------------------------------
    if "REVIEWFLOW" in p:
        if "collect_review_context" not in names:
            return "collect_review_context", {"source": "staged"}, "call_collect", None
        return None, None, None, REVIEWER_JSON

    if "UTFLOW_FAIL" in p:
        if "analyze_test_project" not in names:
            return "analyze_test_project", {}, "call_analyze_fail", None
        if "write_test_file" not in names:
            return "write_test_file", {"path": FAIL_FLOW_TEST_PATH, "content": FAIL_FLOW_TEST_CONTENT}, "call_write_fail", None
        if "run_project_test" not in names:
            return "run_project_test", {"buildSystem": "maven", "testClass": FAIL_FLOW_TEST_CLASS}, "call_run_fail", None
        return None, None, None, unit_test_conclusion(messages, "call_run_fail")

    if "UTFLOW" in p:
        if "analyze_test_project" not in names:
            return "analyze_test_project", {}, "call_analyze", None
        if "write_test_file" not in names:
            return "write_test_file", {"path": FLOW_TEST_PATH, "content": FLOW_TEST_CONTENT}, "call_write", None
        if "run_project_test" not in names:
            return "run_project_test", {"buildSystem": "maven", "testClass": FLOW_TEST_CLASS}, "call_run", None
        return None, None, None, unit_test_conclusion(messages, "call_run")

    # --- Single-shot tools: after a tool result, close the loop -------------
    if last_is_tool:
        return None, None, None, None

    # --- Enterprise Custom Tool single-shot (whitelist proof) ---------------
    # Checked before the native WRITE keyword because the tool names themselves
    # contain "WRITE"/"RUN" substrings.
    if "COLLECT_REVIEW_CONTEXT" in p:
        return "collect_review_context", {"source": "staged"}, "call_collect", None
    if "WRITE_TEST_FILE" in p:
        return "write_test_file", {"path": WHITELIST_TEST_PATH, "content": WHITELIST_TEST_CONTENT}, "call_write", None
    if "RUN_PROJECT_TEST" in p:
        return "run_project_test", {"buildSystem": "maven"}, "call_run", None
    if "WRITE_DOCUMENT" in p:
        return "write_document", {"path": "docs/smoke.md", "content": "smoke\n"}, "call_doc", None

    # --- Native tools --------------------------------------------------------
    if "READ" in p:
        return "read", {"filePath": os.path.join(SMOKE_DIR, "read-me.txt")}, "call_read", None
    if "WRITE" in p:
        return "write", {"filePath": os.path.join(SMOKE_DIR, "write-out.txt"), "content": "smoke-write-ok\n"}, "call_write", None
    if "EDIT" in p:
        return "edit", {
            "filePath": os.path.join(SMOKE_DIR, "edit-me.txt"),
            "oldString": "before",
            "newString": "after",
        }, "call_edit", None
    if "BASH" in p:
        return "bash", {"command": "echo smoke-bash-ok", "description": "smoke echo"}, "call_bash", None
    if "SUBAGENT" in p:
        return "task", {
            "description": "find go files",
            "prompt": "list the go files in this project and report the count",
            "subagent_type": "explore",
        }, "call_subagent", None
    if "SKILL" in p:
        return "skill", {"name": "smoke-skill"}, "call_skill", None

    return None, None, None, None


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, fmt, *args):
        print(fmt % args, file=sys.stderr, flush=True)

    def do_GET(self):
        if self.path == "/v1/models":
            self.send_json({"object": "list", "data": [{"id": "fake-parity", "object": "model"}]})
            return
        self.send_error(404)

    def do_POST(self):
        if self.path != "/v1/chat/completions":
            self.send_error(404)
            return
        length = int(self.headers.get("Content-Length", "0"))
        request = json.loads(self.rfile.read(length))
        print(json.dumps({"path": self.path, "request": request}, separators=(",", ":")), flush=True)

        messages = request.get("messages", [])
        last_user = ""
        for m in reversed(messages):
            if m.get("role") == "user" and isinstance(m.get("content"), str):
                last_user = m["content"]
                break

        tool_name, arguments, call_id, final_text = decide(last_user, messages)

        if tool_name is None:
            chunks = [
                self.chunk({"role": "assistant"}, None),
                self.chunk({"content": final_text if final_text is not None else "smoke-done"}, None),
                self.chunk({}, "stop", usage=True),
            ]
        else:
            chunks = [
                self.chunk({"role": "assistant"}, None),
                self.chunk({"tool_calls": [tool_call(tool_name, "", call_id)]}, None),
                self.chunk({"tool_calls": [{"index": 0, "function": {"arguments": json.dumps(arguments, separators=(",", ":"))}}]}, None),
                self.chunk({}, "tool_calls", usage=True),
            ]

        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.send_header("Connection", "close")
        self.end_headers()
        for chunk in chunks:
            self.wfile.write(("data: " + json.dumps(chunk, separators=(",", ":")) + "\n\n").encode())
            self.wfile.flush()
        self.wfile.write(b"data: [DONE]\n\n")
        self.wfile.flush()
        self.close_connection = True

    def chunk(self, delta, finish_reason, usage=False):
        value = {
            "id": "chatcmpl-fake-parity",
            "object": "chat.completion.chunk",
            "created": int(time.time()),
            "model": "fake-parity",
            "choices": [{"index": 0, "delta": delta, "finish_reason": finish_reason}],
        }
        if usage:
            value["usage"] = {"prompt_tokens": 20, "completion_tokens": 8, "total_tokens": 28}
        return value

    def send_json(self, value):
        payload = json.dumps(value, separators=(",", ":")).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)


if __name__ == "__main__":
    port = int(os.environ.get("FAKE_MODEL_PORT", "49220"))
    server = ThreadingHTTPServer(("127.0.0.1", port), Handler)
    print(f"fake parity model listening on 127.0.0.1:{port}", file=sys.stderr, flush=True)
    server.serve_forever()
