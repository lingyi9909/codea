import type { ToolErrorCategory } from "./types";

// Single error type shared by all 7 tools. A category is always set so the
// failure-classifier and audit log can reason about failures without string
// matching, and so callers can map a category to a stable, actionable message.

export class ToolError extends Error {
  readonly category: ToolErrorCategory;
  override readonly cause?: unknown;

  constructor(category: ToolErrorCategory, message: string, cause?: unknown) {
    super(message);
    this.name = "ToolError";
    this.category = category;
    this.cause = cause;
  }

  toJSON(): { category: ToolErrorCategory; message: string } {
    return { category: this.category, message: this.message };
  }
}

export function invalidInput(message: string): ToolError {
  return new ToolError("INVALID_INPUT", message);
}

export function pathViolation(message: string): ToolError {
  return new ToolError("PATH_VIOLATION", message);
}

export function permissionDenied(message: string): ToolError {
  return new ToolError("PERMISSION_DENIED", message);
}

export function dlpBlocked(message: string): ToolError {
  return new ToolError("DLP_BLOCKED", message);
}

export function timeoutError(message: string): ToolError {
  return new ToolError("TIMEOUT", message);
}

export function commandFailed(message: string, cause?: unknown): ToolError {
  return new ToolError("COMMAND_FAILED", message, cause);
}

export function parseFailed(message: string): ToolError {
  return new ToolError("PARSE_FAILED", message);
}

export function notSupported(message: string): ToolError {
  return new ToolError("NOT_SUPPORTED", message);
}

export function internalError(message: string, cause?: unknown): ToolError {
  return new ToolError("INTERNAL_ERROR", message, cause);
}
