import * as fs from "node:fs";
import * as path from "node:path";
import { redact } from "./security/dlp";
import { toRelativePath } from "./security/path-policy";

// Structured audit log. Records only bounded metadata; never full source, prompt,
// tool output, tokens or absolute user paths. Secret values are redacted before
// write. A write failure is reported but never crashes the agent workflow.

export interface AuditEntry {
  timestamp: string;
  sessionId: string;
  agent: string;
  tool: string;
  action: string;
  result: string;
  duration: number;
  relativePath?: string;
  errorCategory?: string;
}

export interface AuditResult {
  ok: boolean;
  error?: string;
}

export class AuditLogger {
  private readonly logPath: string;
  private readonly projectRoot: string;

  constructor(logPath: string, projectRoot: string) {
    this.logPath = logPath;
    this.projectRoot = projectRoot;
  }

  log(entry: AuditEntry): AuditResult {
    try {
      const sanitized = this.sanitize(entry);
      fs.appendFileSync(this.logPath, JSON.stringify(sanitized) + "\n");
      return { ok: true };
    } catch (err) {
      return { ok: false, error: `audit-write-failed: ${(err as Error).message}` };
    }
  }

  private sanitize(entry: AuditEntry): AuditEntry {
    const out: AuditEntry = {
      timestamp: entry.timestamp,
      sessionId: redact(entry.sessionId, "audit"),
      agent: entry.agent,
      tool: entry.tool,
      action: entry.action,
      result: entry.result,
      duration: entry.duration,
      errorCategory: entry.errorCategory,
    };
    if (entry.relativePath) {
      out.relativePath = this.toProjectRelative(entry.relativePath);
    }
    return out;
  }

  private toProjectRelative(p: string): string {
    const rootAbs = path.resolve(this.projectRoot);
    const target = path.resolve(rootAbs, p);
    return path.relative(rootAbs, target).replace(/\\/g, "/");
  }
}

export { toRelativePath };
