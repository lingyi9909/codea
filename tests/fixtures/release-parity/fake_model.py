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
        print(
            json.dumps(
                {
                    "kind": "release-parity-request",
                    "prompt": prompt,
                    "lastRole": last_role,
                    "stream": bool(req.get("stream")),
                    "tools": len(req.get("tools") or []),
                },
                separators=(",", ":"),
            ),
            file=sys.stderr,
            flush=True,
        )

        chunks = [self.chunk({"role": "assistant"}, None)]
        if last_role == "tool":
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
        elif "APPROVAL TEST" in prompt or "REJECT TEST" in prompt:
            chunks.extend(self.tool_chunks("bash", {"command": "echo release-parity", "description": "release parity approval"}, "call_bash"))
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
