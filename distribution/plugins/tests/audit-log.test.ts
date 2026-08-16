import { afterAll, beforeAll, describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { AuditLogger } from "../src/audit-log";

let tmp: string;
let root: string;

beforeAll(() => {
  tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-audit-"));
  root = path.join(tmp, "project");
  fs.mkdirSync(root, { recursive: true });
});

afterAll(() => {
  fs.rmSync(tmp, { recursive: true, force: true });
});

function readLines(p: string): any[] {
  return fs.readFileSync(p, "utf8").trim().split("\n").map((l) => JSON.parse(l));
}

describe("AuditLogger", () => {
  test("writes a sanitized structured line", () => {
    const logPath = path.join(tmp, "audit.log");
    const logger = new AuditLogger(logPath, root);
    const r = logger.log({
      timestamp: "2026-08-16T00:00:00Z",
      sessionId: "sess-1",
      agent: "code-reviewer",
      tool: "collect_review_context",
      action: "read",
      result: "success",
      duration: 12,
    });
    expect(r.ok).toBe(true);
    const lines = readLines(logPath);
    expect(lines).toHaveLength(1);
    expect(lines[0].tool).toBe("collect_review_context");
    expect(lines[0].sessionId).toBe("sess-1");
  });

  test("redacts secrets in fields", () => {
    const logPath = path.join(tmp, "audit-secret.log");
    const logger = new AuditLogger(logPath, root);
    logger.log({
      timestamp: "2026-08-16T00:00:00Z",
      sessionId: "password=hunter2",
      agent: "a",
      tool: "t",
      action: "x",
      result: "success",
      duration: 0,
    });
    const lines = readLines(logPath);
    expect(JSON.stringify(lines)).not.toContain("hunter2");
  });

  test("converts absolute path to project-relative", () => {
    const logPath = path.join(tmp, "audit-path.log");
    const logger = new AuditLogger(logPath, root);
    logger.log({
      timestamp: "2026-08-16T00:00:00Z",
      sessionId: "s",
      agent: "a",
      tool: "t",
      action: "write",
      result: "success",
      duration: 0,
      relativePath: path.join(root, "src", "main", "java", "A.java"),
    });
    const lines = readLines(logPath);
    expect(lines[0].relativePath).toBe("src/main/java/A.java");
  });

  test("write failure returns ok=false instead of throwing", () => {
    const logger = new AuditLogger(path.join(tmp, "no-such-dir", "audit.log"), root);
    const r = logger.log({
      timestamp: "2026-08-16T00:00:00Z",
      sessionId: "s",
      agent: "a",
      tool: "t",
      action: "x",
      result: "success",
      duration: 0,
    });
    expect(r.ok).toBe(false);
    expect(r.error).toContain("audit-write-failed");
  });
});
