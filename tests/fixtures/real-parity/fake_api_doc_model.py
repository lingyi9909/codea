#!/usr/bin/env python3
"""Deterministic OpenAI-compatible model for Task 16 real-runtime E2E.

The model only decides the next tool call. OpenCode v1.18.11, Codea's plugin,
permission enforcement, tool execution, SSE delivery, and document persistence
remain real. APIDOCFLOW deliberately consumes the real extract_api_spec result,
feeds that exact spec into validate_api_example, then creates the Task 29 plan
required for mutation before rendering Markdown from the structured result and
calling write_document.
"""
import json
import os
import sys
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

SMOKE_DIR = os.environ.get("SMOKE_DIR", "/tmp")
CONTROLLER = "src/main/java/com/example/demo/DemoController.java"
DOC_PATH = "docs/api-demo.md"
EXAMPLE = {"name": "Ada", "email": "ada@example.com", "age": 30}
ALLOWED_PROVENANCE = {"DECLARED", "REFERENCED", "INFERRED"}


def tool_call(tool_name, arguments, call_id):
    return {
        "id": call_id,
        "type": "function",
        "function": {"name": tool_name, "arguments": arguments},
    }


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


def api_doc_plan(goal):
    """Return the smallest valid Task 29 plan needed by document mutations."""
    return {
        "goal": goal,
        "steps": [
            {"id": "inspect", "title": "Inspect the API source"},
            {"id": "document", "title": "Write the API documentation"},
            {
                "id": "verify",
                "title": "Verify the generated documentation",
                "verification": "Use the structured API extraction and validation results",
            },
        ],
    }


def decode_json(value):
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


def find_shape(value, predicate):
    decoded = decode_json(value)
    if decoded is not None and decoded is not value:
        return find_shape(decoded, predicate)
    if isinstance(value, dict):
        if predicate(value):
            return value
        for child in value.values():
            found = find_shape(child, predicate)
            if found is not None:
                return found
    elif isinstance(value, list):
        for child in value:
            found = find_shape(child, predicate)
            if found is not None:
                return found
    return None


def tool_result(messages, call_id, predicate):
    for message in reversed(messages or []):
        if message.get("role") != "tool":
            continue
        msg_call = message.get("tool_call_id")
        if msg_call and msg_call != call_id:
            continue
        found = find_shape(message.get("content"), predicate)
        if found is not None:
            return found
    return None


def api_spec(messages):
    return tool_result(
        messages,
        "call_api_extract",
        lambda value: all(k in value for k in ("controllerName", "basePath", "endpoints", "dtos", "enums")),
    )


def validation_result(messages):
    return tool_result(
        messages,
        "call_api_validate",
        lambda value: all(k in value for k in ("valid", "errors", "warnings")),
    )


def post_endpoint_index(spec):
    for index, endpoint in enumerate(spec.get("endpoints") or []):
        if endpoint.get("method") == "POST" and endpoint.get("requestBody"):
            return index
    return 0


def render_markdown(spec, validation):
    lines = [
        "# API Documentation",
        "",
        f"Controller: {spec.get('controllerName') or 'Not determined from code'}",
        f"Base Path: {spec.get('basePath') or 'Not determined from code'}",
        "",
        "Description: Not determined from code",
        "",
        "## Endpoints",
    ]
    provenance = []
    for endpoint in spec.get("endpoints") or []:
        method = endpoint.get("method") or "Not determined from code"
        path = endpoint.get("path") or "Not determined from code"
        lines.append(f"- {method} {path}")
        request_body = endpoint.get("requestBody") or {}
        if request_body:
            lines.append(f"  - Request DTO: {request_body.get('type') or 'Not determined from code'}")
        for error in endpoint.get("errorCodes") or []:
            source = error.get("source") or ""
            if source not in ALLOWED_PROVENANCE:
                raise ValueError(f"unexpected error provenance: {source}")
            provenance.append((error.get("code") or "Not determined from code", source))

    lines += ["", "## DTO / Validation"]
    for dto_name, dto in sorted((spec.get("dtos") or {}).items()):
        lines.append(f"### {dto_name}")
        for field in dto.get("fields") or []:
            validations = field.get("validation") or []
            suffix = " ".join(validations) if validations else "Not determined from code"
            lines.append(f"- {field.get('name')}: {field.get('type')} [{suffix}]")

    lines += ["", "## Error Codes"]
    if provenance:
        seen = set()
        for code, source in provenance:
            key = (code, source)
            if key in seen:
                continue
            seen.add(key)
            lines.append(f"- {code} | {source} | Meaning: Not determined from code")
    else:
        lines.append("- Not determined from code")

    valid = bool(validation and validation.get("valid") is True)
    lines += [
        "",
        "## Example",
        "```json",
        json.dumps(EXAMPLE, ensure_ascii=False, sort_keys=True),
        "```",
        f"Example validation: {'PASS' if valid else 'FAIL'}",
        "",
    ]
    return "\n".join(lines)


def decide(prompt, messages):
    text = (prompt or "").upper()
    names = assistant_tool_names(messages)
    last_is_tool = bool(messages) and messages[-1].get("role") == "tool"

    if "APIDOCFLOW" in text:
        if "extract_api_spec" not in names:
            return "extract_api_spec", {"controllerFile": CONTROLLER}, "call_api_extract", None
        spec = api_spec(messages)
        if spec is None:
            return None, None, None, json.dumps({"result": "FAIL", "reason": "extract_api_spec result missing"})
        if "validate_api_example" not in names:
            return "validate_api_example", {
                "example": EXAMPLE,
                "spec": spec,
                "endpointIndex": post_endpoint_index(spec),
            }, "call_api_validate", None
        validation = validation_result(messages)
        if validation is None:
            return None, None, None, json.dumps({"result": "FAIL", "reason": "validate_api_example result missing"})
        if "task_plan" not in names:
            return "task_plan", api_doc_plan("Generate validated API documentation"), "call_api_plan", None
        if "write_document" not in names:
            return "write_document", {
                "path": DOC_PATH,
                "content": render_markdown(spec, validation),
            }, "call_api_write", None
        return None, None, None, json.dumps({"result": "PASS", "source": "write_document"}, separators=(",", ":"))

    # WRITE_DOCUMENT is a mutating custom-tool scenario. Keep it ahead of the
    # generic last-tool close so a successful task_plan can advance to the write.
    if "WRITE_DOCUMENT" in text:
        if "task_plan" not in names:
            return "task_plan", api_doc_plan("Write a deterministic API document"), "call_doc_plan", None
        if "write_document" not in names:
            return "write_document", {"path": "docs/smoke.md", "content": "smoke\n"}, "call_doc", None
        return None, None, None, None

    if last_is_tool:
        return None, None, None, None
    if "WRITE_TEST_FILE" in text:
        return "write_test_file", {
            "path": "src/test/java/com/example/demo/ShouldBeDenied.java",
            "content": "class ShouldBeDenied {}\n",
        }, "call_denied", None
    if "READ" in text:
        return "read", {"filePath": os.path.join(SMOKE_DIR, "read-me.txt")}, "call_read", None
    return None, None, None, None


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, fmt, *args):
        print(fmt % args, file=sys.stderr, flush=True)

    def do_GET(self):
        if self.path == "/v1/models":
            self.send_json({"object": "list", "data": [{"id": "fake-api-doc", "object": "model"}]})
            return
        self.send_error(404)

    def do_POST(self):
        if self.path != "/v1/chat/completions":
            self.send_error(404)
            return
        length = int(self.headers.get("Content-Length", "0"))
        request = json.loads(self.rfile.read(length))
        messages = request.get("messages", [])
        last_user = ""
        for message in reversed(messages):
            if message.get("role") == "user" and isinstance(message.get("content"), str):
                last_user = message["content"]
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
            "id": "chatcmpl-fake-api-doc",
            "object": "chat.completion.chunk",
            "created": int(time.time()),
            "model": "fake-api-doc",
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
    port = int(os.environ.get("FAKE_MODEL_PORT", "49221"))
    server = ThreadingHTTPServer(("127.0.0.1", port), Handler)
    print(f"fake api-doc model listening on 127.0.0.1:{port}", file=sys.stderr, flush=True)
    server.serve_forever()
