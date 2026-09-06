import { afterAll, beforeAll, describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { AuditLogger } from "../src/audit-log";
import { RuntimeSecurityGuard } from "../src/runtime-security-guard";
import type { PermissionsConfig } from "../src/permissions";

const LEVELS = ["weak", "medium", "strong"] as const;
const PERMISSIONS_PATH = path.resolve(import.meta.dir, "../../config/opencode/permissions.json");
const EVALUATOR_MANIFEST = path.resolve(import.meta.dir, "../../agents/model-evaluator/manifest.yaml");

let tmp: string;
let root: string;

beforeAll(() => {
  tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-task32-security-"));
  root = path.join(tmp, "project");
  fs.mkdirSync(root, { recursive: true });
});

afterAll(() => fs.rmSync(tmp, { recursive: true, force: true }));

function guard(level: (typeof LEVELS)[number]): RuntimeSecurityGuard {
  // Capability level is deliberately not passed into the security boundary.
  // The test loops every strategy level to make that architectural invariant explicit.
  const log = path.join(tmp, `${level}-${Math.random().toString(36).slice(2)}.log`);
  return new RuntimeSecurityGuard(new AuditLogger(log, root));
}

function snapshot(level: (typeof LEVELS)[number]) {
  const base = { sessionId: `s-${level}`, agent: "general", tool: "bash", action: "execute", projectRoot: root };
  const g = guard(level);
  const permissions = JSON.parse(fs.readFileSync(PERMISSIONS_PATH, "utf8")) as PermissionsConfig;
  return {
    deniedCommand: g.before({ ...base, command: "rm -rf /" }).decision,
    networkCommand: g.before({ ...base, command: "curl http://example.com" }).decision,
    pathEscape: g.before({ ...base, action: "write", tool: "write", targetPath: "../../outside.txt" }).decision,
    dlpSecret: g.before({ ...base, action: "write", tool: "write", input: { content: "password=supersecret" } }).decision,
    approvalWrite: permissions.agents.general?.write,
  };
}

function manifestTools(): Map<string, string> {
  const manifest = fs.readFileSync(EVALUATOR_MANIFEST, "utf8");
  const tools = new Map<string, string>();
  let inTools = false;
  for (const line of manifest.split(/\r?\n/)) {
    if (line === "tools:") { inTools = true; continue; }
    if (!inTools) continue;
    if (line !== "" && !line.startsWith("  ")) break;
    const match = /^  ([A-Za-z0-9_-]+):\s*(allow|ask|deny)\s*$/.exec(line);
    if (match) tools.set(match[1]!, match[2]!);
  }
  return tools;
}

describe("Task 32 strategy security invariance", () => {
  test("weak, medium, and strong preserve identical security decisions", () => {
    const expected = {
      deniedCommand: "deny",
      networkCommand: "deny",
      pathEscape: "deny",
      dlpSecret: "deny",
      approvalWrite: "ask",
    };
    for (const level of LEVELS) expect(snapshot(level)).toEqual(expected);
  });

  test("security implementation has no model-strategy dependency", () => {
    const files = [
      "../src/runtime-security-guard.ts",
      "../src/security/command-policy.ts",
      "../src/security/path-policy.ts",
      "../src/security/dlp.ts",
      "../src/permissions.ts",
    ];
    for (const relative of files) {
      const source = fs.readFileSync(path.resolve(import.meta.dir, relative), "utf8").toLowerCase();
      expect(source).not.toContain("model-strategy");
      expect(source).not.toContain("modelprofile");
      expect(source).not.toContain("capabilitylevel");
    }
  });

  test("strong profile cannot expand model-evaluator beyond probe tools", () => {
    const tools = manifestTools();
    const allowed = [...tools.entries()].filter(([, decision]) => decision === "allow").map(([name]) => name).sort();
    expect(allowed).toEqual(["probe_patch", "probe_plan", "probe_structured", "probe_tool_call"]);
    for (const tool of ["read", "grep", "glob", "write", "edit", "bash", "run_project_test", "verify_project", "dify-query"]) {
      expect(tools.get(tool)).toBe("deny");
    }
  });
});
