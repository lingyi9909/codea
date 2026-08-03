#!/usr/bin/env python3
import json
import sys
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, fmt, *args):
        print(fmt % args, file=sys.stderr, flush=True)

    def do_GET(self):
        if self.path == "/v1/models":
            self.send_json({"object": "list", "data": [{"id": "fake-reasoning", "object": "model"}]})
            return
        self.send_error(404)

    def do_POST(self):
        if self.path != "/v1/chat/completions":
            self.send_error(404)
            return
        length = int(self.headers.get("Content-Length", "0"))
        request = json.loads(self.rfile.read(length))
        print(json.dumps({"path": self.path, "stream": request.get("stream"), "model": request.get("model")}, separators=(",", ":")), flush=True)
        chunks = [
            self.chunk({"role": "assistant"}, None),
            self.chunk({"reasoning_content": "considering options"}, None),
            self.chunk({"content": "final answer"}, None),
            self.chunk({}, "stop", usage=True),
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

    @staticmethod
    def chunk(delta, finish_reason, usage=False):
        value = {
            "id": "chatcmpl-codea-s4",
            "object": "chat.completion.chunk",
            "created": int(time.time()),
            "model": "fake-reasoning",
            "choices": [{"index": 0, "delta": delta, "finish_reason": finish_reason}],
        }
        if usage:
            value["usage"] = {"prompt_tokens": 12, "completion_tokens": 5, "total_tokens": 17}
        return value

    def send_json(self, value):
        payload = json.dumps(value, separators=(",", ":")).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)


if __name__ == "__main__":
    server = ThreadingHTTPServer(("127.0.0.1", 49222), Handler)
    print("fake reasoning server listening on 127.0.0.1:49222", file=sys.stderr, flush=True)
    server.serve_forever()
