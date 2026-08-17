import { AuditLogger, type AuditEntry } from "./audit-log";
import { analyzeCommand } from "./security/command-policy";
import { scanDlp } from "./security/dlp";
import type { DlpContext } from "./security/types";
import { resolveProjectPath } from "./security/path-policy";
import { RiskAsk, RiskDeny } from "./security/types";

// Unified before/after hook. Applies path policy, command policy and DLP input
// on the way in; DLP output + audit on the way out. Contains no tool-specific
// business logic — that stays in the individual tools.

export type GuardDecision = "allow" | "ask" | "deny";

export interface BeforeInput {
  sessionId: string;
  agent: string;
  tool: string;
  action: string;
  projectRoot: string;
  input?: unknown;
  command?: string;
  targetPath?: string;
}

export interface GuardResult {
  decision: GuardDecision;
  reason?: string;
  redactedInput?: unknown;
}

export interface AfterInput {
  sessionId: string;
  agent: string;
  tool: string;
  action: string;
  projectRoot: string;
  targetPath?: string;
  durationMs: number;
  ok: boolean;
  output?: unknown;
  errorCategory?: string;
}

const WRITE_ACTIONS = new Set(["write", "create", "overwrite", "append", "edit"]);

function stringify(value: unknown): string {
  if (typeof value === "string") return value;
  if (value === undefined || value === null) return "";
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

export class RuntimeSecurityGuard {
  private readonly audit: AuditLogger;

  constructor(audit: AuditLogger) {
    this.audit = audit;
  }

  before(input: BeforeInput): GuardResult {
    // 1. path policy
    if (input.targetPath && WRITE_ACTIONS.has(input.action)) {
      try {
        resolveProjectPath(input.projectRoot, input.targetPath);
      } catch (err) {
        this.auditDeny(input, `path-violation: ${(err as Error).message}`);
        return { decision: "deny", reason: `path-violation: ${(err as Error).message}` };
      }
    }

    // 2. command policy
    if (input.command !== undefined) {
      const analysis = analyzeCommand(input.command);
      if (analysis.risk === RiskDeny) {
        this.auditDeny(input, `command-denied: ${analysis.matchedRule}`);
        return { decision: "deny", reason: `command-denied: ${analysis.matchedRule}` };
      }
      if (analysis.risk === RiskAsk) {
        this.auditDeny(input, "command-requires-approval");
        return { decision: "ask", reason: "command-requires-approval" };
      }
      // RiskSafe falls through — a safe command must still be DLP-scanned so a
      // secret embedded in its arguments is not skipped.
    }

    // 3. DLP input — scan the command string (if present) together with the tool
    // input so safe commands cannot bypass secret/path redaction.
    const dlpParts: string[] = [];
    if (input.command !== undefined && input.command !== "") dlpParts.push(input.command);
    const inputStr = stringify(input.input);
    if (inputStr !== "") dlpParts.push(inputStr);
    const dlp = scanDlp(dlpParts.join(" "), "tool-input");
    if (!dlp.allowed) {
      const rule = dlp.findings[0]?.rule ?? "secret";
      this.auditDeny(input, `dlp-blocked: ${rule}`);
      return { decision: "deny", reason: `dlp-blocked: ${rule}` };
    }

    return { decision: "allow", redactedInput: this.redactValue(input.input, "tool-input") };
  }

  after(input: AfterInput): void {
    const dlp = scanDlp(stringify(input.output), "tool-output");
    const result = !input.ok ? "error" : dlp.allowed ? "success" : "dlp-blocked";
    const errorCategory = input.ok ? (dlp.allowed ? input.errorCategory : "DLP_BLOCKED") : input.errorCategory;
    const entry: AuditEntry = {
      timestamp: new Date().toISOString(),
      sessionId: input.sessionId,
      agent: input.agent,
      tool: input.tool,
      action: input.action,
      result,
      duration: input.durationMs,
      relativePath: input.targetPath,
      errorCategory,
    };
    this.audit.log(entry);
  }

  private auditDeny(input: BeforeInput, reason: string): void {
    const entry: AuditEntry = {
      timestamp: new Date().toISOString(),
      sessionId: input.sessionId,
      agent: input.agent,
      tool: input.tool,
      action: input.action,
      result: "denied",
      duration: 0,
      relativePath: input.targetPath,
      errorCategory: reason,
    };
    this.audit.log(entry);
  }

  private redactValue(value: unknown, context: DlpContext): unknown {
    if (typeof value === "string") return scanDlp(value, context).redacted;
    if (Array.isArray(value)) return value.map((v) => this.redactValue(v, context));
    if (value && typeof value === "object") {
      const out: Record<string, unknown> = {};
      for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
        out[k] = this.redactValue(v, context);
      }
      return out;
    }
    return value;
  }
}
