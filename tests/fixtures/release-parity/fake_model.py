#!/usr/bin/env python3
"""Deterministic OpenAI-compatible model for the Task 21 dual-runtime parity gate."""
import json
import os
import sys
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

SMOKE_DIR = os.environ.get("SMOKE_DIR", "/tmp")


def tool_call(name, args, call_id):
    return {
        "id": call_id,
        "type": "function",
        "function": {"name": name, "arguments": args},
    }


def last_user(messages):
    for message in reversed(messages or []):
        if message.get("role") == "user" and isinstance(message.get("content"), str):
            return message["content"]
    return ""


def advertised_tool_names(tools):
    names = set()
    for tool in tools or []:
        function = tool.get("function") or {}
        name = function.get("name")
        if name:
            names.add(name)
    return names


def assistant_tool_names(messages):
    names = []
    for message in messages or []:
        if message.get("role") != "assistant":
            continue
        for call in message.get("tool_calls") or []:
            name = (call.get("function") or {}).get("name")
            if name:
                names.append(name)
    return names


def approval_plan(prompt):
    action = "Execute the approval-gated command"
    if "REJECT" in (prompt or "").upper():
        action = "Attempt the approval-gated command and preserve rejection"
    return {
        "goal": action,
        "steps": [
            {"id": "prepare", "title": "Prepare the approval-gated command"},
            {"id": "execute", "title": action},
            {
                "id": "verify",
                "title": "Verify the runtime approval outcome",
                "verification": "Use the runtime approval and tool result events",
            },
        ],
    }


def decide(prompt, messages, tools):
    """Choose the next parity action without changing baseline semantics.

    Vanilla OpenCode does not advertise Codea's task_plan tool, so the baseline
    continues to call bash directly and exercises OpenCode's approval flow.
    The Codea candidate advertises task_plan and enforces Task 29's plan gate,
    so it creates a valid plan first and only then calls the same bash tool.
    """
    p = (prompt or "").upper()
    available = advertised_tool_names(tools)
    names = assistant_tool_names(messages)

    if "APPROVAL TEST" in p or "REJECT TEST" in p:
        if "task_plan" in available and "task_plan" not in names:
            call_id = "call_plan_reject" if "REJECT TEST" in p else "call_plan_approval"
            return "task_plan", approval_plan(p), call_id, None
        if "bash" not in names:
            return (
                "bash",
                {"command": "echo release-parity", "description": "release parity approval"},
                "call_bash",
                None,
            )
        return None, None, None, "tool-done"

    return None, None, None, None


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, fmt, *args):
        print(fmt % args, file=sys.stderr, flush=True)

    def do_GET(self):
        if self.path == "/v1/models":
            self.send_json({"object": "list", "data": [{"id": "release-parity", "object": "model"}]})
            return
        self.send_error(404)

    def do_POST(self):
        if self.path != "/v1/chat/completions":
            self.send_error(404)
            return
        length = int(self.headers.get("Content-Length", "0"))
        req = json.loads(self.rfile.read(length))
        messages = req.get("messages", [])
        prompt = last_user(messages).upper()
        last_role = messages[-1].get("role") if messages else ""
        tools = req.get("tools") or []
        print(
            json.dumps(
                {
                    "kind": "release-parity-request",
                    "prompt": prompt,
                    "lastRole": last_role,
                    "stream": bool(req.get("stream")),
                    "tools": len(tools),
                },
                separators=(",", ":"),
            ),
            file=sys.stderr,
            flush=True,
        )

        chunks = [self.chunk({"role": "assistant"}, None)]
        if "APPROVAL TEST" in prompt or "REJECT TEST" in prompt:
            tool_name, arguments, call_id, final_text = decide(prompt, messages, tools)
            if tool_name is not None:
                chunks.extend(self.tool_chunks(tool_name, arguments, call_id))
            else:
                chunks.append(self.chunk({"content": final_text or "tool-done"}, None))
                chunks.append(self.chunk({}, "stop", usage=True))
        elif last_role == "tool":
            chunks.append(self.chunk({"content": "tool-done"}, None))
            chunks.append(self.chunk({}, "stop", usage=True))
        elif "REASONING TEST" in prompt:
            # OpenAI-compatible providers expose reasoning via the
            # reasoning_content delta; OpenCode v1.18.11 maps it to a
            # reasoning part when the provider/model capabilities allow it.
            chunks.append(self.chunk({"reasoning_content": "parity-thinking"}, None))
            chunks.append(self.chunk({"content": "parity-answer"}, None))
            chunks.append(self.chunk({}, "stop", usage=True))
        elif "TOOL TEST" in prompt:
            chunks.extend(self.tool_chunks("read", {"filePath": os.path.join(SMOKE_DIR, "read-me.txt")}, "call_read"))
        else:
            chunks.append(self.chunk({"content": "parity-answer"}, None))
            chunks.append(self.chunk({}, "stop", usage=True))

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

    def tool_chunks(self, name, arguments, call_id):
        return [
            self.chunk({"tool_calls": [tool_call(name, "", call_id)]}, None),
            self.chunk({"tool_calls": [{"index": 0, "function": {"arguments": json.dumps(arguments, separators=(",", ":"))}}]}, None),
            self.chunk({}, "tool_calls", usage=True),
        ]

    def chunk(self, delta, finish_reason, usage=False):
        out = {
            "id": "chatcmpl-release-parity",
            "object": "chat.completion.chunk",
            "created": int(time.time()),
            "model": "release-parity",
            "choices": [{"index": 0, "delta": delta, "finish_reason": finish_reason}],
        }
        if usage:
            out["usage"] = {"prompt_tokens": 16, "completion_tokens": 8, "total_tokens": 24}
        return out

    def send_json(self, value):
        body = json.dumps(value, separators=(",", ":")).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


if __name__ == "__main__":
    port = int(os.environ.get("FAKE_MODEL_PORT", "49240"))
    server = ThreadingHTTPServer(("127.0.0.1", port), Handler)
    print(f"release parity fake model listening on 127.0.0.1:{port}", flush=True)
    server.serve_forever()
