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
            self.send_json({"object": "list", "data": [{"id": "fake-chat", "object": "model"}]})
            return
        self.send_error(404)

    def do_POST(self):
        if self.path != "/v1/chat/completions":
            self.send_error(404)
            return

        length = int(self.headers.get("Content-Length", "0"))
        request = json.loads(self.rfile.read(length))
        print(json.dumps({"path": self.path, "request": request}, separators=(",", ":")), flush=True)

        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.send_header("Connection", "close")
        self.end_headers()

        chunks = [
            {
                "id": "chatcmpl-codea-s2",
                "object": "chat.completion.chunk",
                "created": int(time.time()),
                "model": "fake-chat",
                "choices": [{"index": 0, "delta": {"role": "assistant"}, "finish_reason": None}],
            },
            {
                "id": "chatcmpl-codea-s2",
                "object": "chat.completion.chunk",
                "created": int(time.time()),
                "model": "fake-chat",
                "choices": [
                    {"index": 0, "delta": {"content": "hello from s2"}, "finish_reason": None}
                ],
            },
            {
                "id": "chatcmpl-codea-s2",
                "object": "chat.completion.chunk",
                "created": int(time.time()),
                "model": "fake-chat",
                "choices": [{"index": 0, "delta": {}, "finish_reason": "stop"}],
                "usage": {"prompt_tokens": 10, "completion_tokens": 3, "total_tokens": 13},
            },
        ]
        for chunk in chunks:
            self.wfile.write(("data: " + json.dumps(chunk, separators=(",", ":")) + "\n\n").encode())
            self.wfile.flush()
        self.wfile.write(b"data: [DONE]\n\n")
        self.wfile.flush()
        self.close_connection = True

    def send_json(self, value):
        payload = json.dumps(value, separators=(",", ":")).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)


if __name__ == "__main__":
    server = ThreadingHTTPServer(("127.0.0.1", 49220), Handler)
    print("fake OpenAI-compatible server listening on 127.0.0.1:49220", file=sys.stderr, flush=True)
    server.serve_forever()
