import { PathViolationError } from "../security/path-policy";
import { ToolError } from "./errors";
import type { ToolErrorCategory } from "./types";

// Maps arbitrary throwables and command results onto the single 9-way category
// taxonomy shared by every tool. Keeps category assignment in one place so the
// audit log and error surfaces stay consistent.

const TIMEOUT_SIGNALS = new Set(["SIGTERM", "SIGKILL", "ETIMEDOUT"]);

export function classifyError(err: unknown): ToolErrorCategory {
  if (err instanceof ToolError) return err.category;
  if (err instanceof PathViolationError) return "PATH_VIOLATION";

  if (err instanceof Error) {
    const anyErr = err as Error & { code?: string; signal?: string; killed?: boolean };
    if (anyErr.killed || (anyErr.signal && TIMEOUT_SIGNALS.has(anyErr.signal)) || anyErr.code === "ETIMEDOUT") {
      return "TIMEOUT";
    }
  }

  return "INTERNAL_ERROR";
}

export function toToolError(err: unknown, fallbackMessage: string): ToolError {
  if (err instanceof ToolError) return err;
  if (err instanceof PathViolationError) return new ToolError("PATH_VIOLATION", err.message, err);
  if (err instanceof Error) return new ToolError(classifyError(err), err.message || fallbackMessage, err);
  return new ToolError("INTERNAL_ERROR", fallbackMessage, err);
}

// Classifies a completed command result (nonzero exit vs timeout). A zero exit
// is not an error, so callers gate on exitCode before reaching here.
export function classifyCommandFailure(exitCode: number | null, timedOut: boolean): ToolErrorCategory {
  if (timedOut) return "TIMEOUT";
  return "COMMAND_FAILED";
}
