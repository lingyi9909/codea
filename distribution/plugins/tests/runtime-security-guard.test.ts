import { afterAll, beforeAll, describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { AuditLogger } from "../src/audit-log";
import { RuntimeSecurityGuard } from "../src/runtime-security-guard";

let tmp: string;
let root: string;

beforeAll(() => {
  tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-guard-"));
  root = path.join(tmp, "project");
  fs.mkdirSync(root, { recursive: true });
});

afterAll(() => {
  fs.rmSync(tmp, { recursive: true, force: true });
});

function guard(): { g: RuntimeSecurityGuard; logPath: string } {
  const logPath = path.join(tmp, `guard-${Math.random().toString(36).slice(2)}.log`);
  return { g: new RuntimeSecurityGuard(new AuditLogger(logPath, root)), logPath };
}

const base = () => ({ sessionId: "s", agent: "code-reviewer", tool: "t", action: "write", projectRoot: root });

describe("RuntimeSecurityGuard.before — path policy", () => {
  test("path escape denies", () => {
    const { g } = guard();
    const r = g.before({ ...base(), targetPath: "../../etc/passwd" });
    expect(r.decision).toBe("deny");
    expect(r.reason).toContain("path-violation");
  });
  test("in-root path allows", () => {
    const { g } = guard();
    const r = g.before({ ...base(), targetPath: "src/main/java/A.java" });
    expect(r.decision).toBe("allow");
  });
});

describe("RuntimeSecurityGuard.before — command policy", () => {
  test("dangerous command denies", () => {
    const { g } = guard();
    const r = g.before({ ...base(), action: "execute", tool: "bash", command: "rm -rf /" });
    expect(r.decision).toBe("deny");
    expect(r.reason).toContain("command-denied");
  });
  test("safe command allows", () => {
    const { g } = guard();
    const r = g.before({ ...base(), action: "execute", tool: "bash", command: "git status" });
    expect(r.decision).toBe("allow");
  });
  test("unknown command asks", () => {
    const { g } = guard();
    const r = g.before({ ...base(), action: "execute", tool: "bash", command: "mystery-tool" });
    expect(r.decision).toBe("ask");
  });
  test("safe command with sensitive path arg is denied (not allowed)", () => {
    const { g } = guard();
    const r = g.before({ ...base(), action: "execute", tool: "bash", command: "cat .env" });
    expect(r.decision).toBe("deny");
    expect(r.reason ?? "").toContain("command-denied");
  });
  test("safe command carrying a secret is DLP-blocked (does not skip DLP)", () => {
    const { g } = guard();
    const r = g.before({ ...base(), action: "execute", tool: "bash", command: "grep password=hunter2 file.txt" });
    expect(r.decision).toBe("deny");
    expect(r.reason ?? "").toContain("dlp-blocked");
  });
});

describe("RuntimeSecurityGuard.before — DLP input", () => {
  test("secret in input denies", () => {
    const { g } = guard();
    const r = g.before({ ...base(), input: { content: "password=supersecret" } });
    expect(r.decision).toBe("deny");
    expect(r.reason).toContain("dlp-blocked");
  });
  test("safe input allows and redacts", () => {
    const { g } = guard();
    const r = g.before({ ...base(), input: { content: "hello world" } });
    expect(r.decision).toBe("allow");
  });
});

describe("RuntimeSecurityGuard.after — audit", () => {
  test("writes an audit entry", () => {
    const { g, logPath } = guard();
    g.after({ ...base(), durationMs: 5, ok: true, output: "done", targetPath: "src/A.java" });
    const content = fs.readFileSync(logPath, "utf8");
    expect(content).toContain("src/A.java");
    const entry = JSON.parse(content.trim().split("\n")[0]);
    expect(entry.result).toBe("success");
    expect(entry.duration).toBe(5);
  });
});
