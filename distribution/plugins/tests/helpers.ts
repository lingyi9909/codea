import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { AuditLogger } from "../src/audit-log";
import { RuntimeSecurityGuard } from "../src/runtime-security-guard";
import type { ToolContext } from "../src/tools/types";

// Shared test setup: a temp project root + a redacting audit log, wired into a
// real RuntimeSecurityGuard so tool tests exercise the full security path.

export function makeTempRoot(prefix = "codea-tool-"): string {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), prefix));
  const root = path.join(tmp, "project");
  fs.mkdirSync(root, { recursive: true });
  return root;
}

export function makeContext(projectRoot: string): { ctx: ToolContext; logPath: string } {
  // Audit log lives in the system temp dir, never next to the fixture, so tests
  // against a checked-in fixture cannot dirty the repo tree.
  const logPath = path.join(os.tmpdir(), `codea-audit-${process.pid}-${Math.random().toString(36).slice(2)}.log`);
  const audit = new AuditLogger(logPath, projectRoot);
  const guard = new RuntimeSecurityGuard(audit);
  const ctx: ToolContext = {
    sessionId: "test-session",
    agent: "unit-test-generator",
    projectRoot,
    audit,
    guard,
  };
  return { ctx, logPath };
}

export function readAudit(logPath: string): string {
  try {
    return fs.readFileSync(logPath, "utf8");
  } catch {
    return "";
  }
}
