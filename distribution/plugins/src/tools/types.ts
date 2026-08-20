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

// WriteOwnership is the server-side record of files a single (session, agent)
// run has created. It is what makes "never overwrite an existing test" a real
// guarantee instead of a prompt instruction: an overwrite is only allowed for a
// canonical path this run actually created, and only when overwrite=true.
export interface WriteOwnership {
  record(absPath: string): void;
  owns(absPath: string): boolean;
}

export interface ToolContext {
  sessionId: string;
  agent: string;
  projectRoot: string;
  audit: AuditLogger;
  guard: RuntimeSecurityGuard;
  ownership?: WriteOwnership;
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
