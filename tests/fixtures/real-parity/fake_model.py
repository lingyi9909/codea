#!/usr/bin/env python3
"""Deterministic OpenAI-compatible streaming model stub for the real-parity smoke.

This is NOT a real LLM. It scripts a fixed tool-call lifecycle so the Codea
real-runtime smoke can exercise OpenCode's native Read/Write/Edit/Bash/Task
(subagent)/Skill tools end-to-end without a network or an API key. The OpenCode
runtime, Agent Loop, message persistence, permission gating and SSE are all
real; only the model is a stub (same methodology as the S2/S3 Phase 0 spikes).

State machine, keyed on the last user message:
  READ      -> emit a `read` tool call on SMOKE_DIR/read-me.txt
  WRITE     -> emit a `write` tool call creating SMOKE_DIR/write-out.txt
  EDIT      -> emit an `edit` tool call on SMOKE_DIR/edit-me.txt
  BASH      -> emit a `bash` tool call (echo smoke-bash-ok)
  SUBAGENT  -> emit a `task` tool call delegating to the `explore` subagent
  SKILL     -> emit a `skill` tool call for `smoke-skill`
  otherwise -> emit a plain text answer (also handles the subagent's own turn)

A request that already carries a `tool` role message (a tool result) always
answers with plain text, closing the tool loop.
"""
import json
import os
import sys
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

SMOKE_DIR = os.environ.get("SMOKE_DIR", "/tmp")


def tool_call(tool_name, arguments, call_id):
    return {
        "id": call_id,
        "type": "function",
        "function": {"name": tool_name, "arguments": arguments},
    }


def script_for(prompt):
    p = (prompt or "").upper()
    if "READ" in p:
        return "read", {"filePath": os.path.join(SMOKE_DIR, "read-me.txt")}, "call_read"
    if "WRITE" in p:
        return "write", {"filePath": os.path.join(SMOKE_DIR, "write-out.txt"), "content": "smoke-write-ok\n"}, "call_write"
    if "EDIT" in p:
        return "edit", {
            "filePath": os.path.join(SMOKE_DIR, "edit-me.txt"),
            "oldString": "before",
            "newString": "after",
        }, "call_edit"
    if "BASH" in p:
        return "bash", {"command": "echo smoke-bash-ok", "description": "smoke echo"}, "call_bash"
    if "SUBAGENT" in p:
        return "task", {
            "description": "find go files",
            "prompt": "list the go files in this project and report the count",
            "subagent_type": "explore",
        }, "call_subagent"
    if "SKILL" in p:
        return "skill", {"name": "smoke-skill"}, "call_skill"
    return None, None, None


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

        # A tool result is the *immediately preceding* message only; a resumed
        # session carries earlier tool results in history but a new user prompt
        # must still script a fresh tool call, not answer text.
        has_tool_result = bool(messages) and messages[-1].get("role") == "tool"
        tool_name, arguments, call_id = (None, None, None)
        if not has_tool_result:
            tool_name, arguments, call_id = script_for(last_user)

        if tool_name is None:
            chunks = [
                self.chunk({"role": "assistant"}, None),
                self.chunk({"content": "smoke-done"}, None),
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
