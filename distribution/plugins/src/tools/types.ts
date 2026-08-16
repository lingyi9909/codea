import type { AuditLogger } from "../audit-log";
import type { RuntimeSecurityGuard } from "../runtime-security-guard";
import type { ToolError } from "./errors";

// Shared tool foundation. Every enterprise tool consumes the same context and
// returns the same result shape so callers (and the security guard) treat all 7
// tools uniformly.

export type ToolErrorCategory =
  | "INVALID_INPUT"
  | "PATH_VIOLATION"
  | "PERMISSION_DENIED"
  | "DLP_BLOCKED"
  | "TIMEOUT"
  | "COMMAND_FAILED"
  | "PARSE_FAILED"
  | "NOT_SUPPORTED"
  | "INTERNAL_ERROR";

export interface ToolContext {
  sessionId: string;
  agent: string;
  projectRoot: string;
  audit: AuditLogger;
  guard: RuntimeSecurityGuard;
}

export type ToolResult<T> =
  | { ok: true; data: T }
  | { ok: false; error: ToolError };

export function ok<T>(data: T): ToolResult<T> {
  return { ok: true, data };
}

export function err<T>(error: ToolError): ToolResult<T> {
  return { ok: false, error };
}
