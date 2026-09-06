import { invalidInput } from "./errors";
import { err, ok, type ToolContext, type ToolResult } from "./types";

export type ProbeName = "tool_call" | "structured" | "patch" | "planning";

export interface ProbeAcknowledgement {
  acknowledgement: "PASS";
  codeaProbe: ProbeName;
  codeaProbeResult: "pass";
  codeaProbeAttempt: "1" | "2";
}

export class ProbeAttemptState {
  private readonly counts = new Map<string, number>();

  next(sessionId: string, probe: ProbeName): 1 | 2 | null {
    const key = `${sessionId}\u0000${probe}`;
    const next = (this.counts.get(key) ?? 0) + 1;
    if (next > 2) return null;
    this.counts.set(key, next);
    return next as 1 | 2;
  }
}

export function exactObject(value: unknown, keys: readonly string[]): value is Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) return false;
  const actual = Object.keys(value as Record<string, unknown>).sort();
  const expected = [...keys].sort();
  return actual.length === expected.length && actual.every((key, index) => key === expected[index]);
}

export function beginProbe(
  state: ProbeAttemptState,
  ctx: ToolContext,
  tool: string,
  probe: ProbeName,
  startedAt: number,
): { attempt: 1 | 2 } | { limited: ToolResult<ProbeAcknowledgement> } {
  const attempt = state.next(ctx.sessionId, probe);
  if (attempt !== null) return { attempt };
  const error = invalidInput(`PROBE_ATTEMPT_LIMIT: ${probe} allows at most two attempts per runtime session`);
  ctx.guard.after({
    sessionId: ctx.sessionId,
    agent: ctx.agent,
    tool,
    action: "probe",
    projectRoot: ctx.projectRoot,
    durationMs: Date.now() - startedAt,
    ok: false,
    errorCategory: error.category,
  });
  return { limited: err(error) };
}

export function rejectProbe(
  ctx: ToolContext,
  tool: string,
  message: string,
  startedAt: number,
): ToolResult<ProbeAcknowledgement> {
  const error = invalidInput(message);
  ctx.guard.after({
    sessionId: ctx.sessionId,
    agent: ctx.agent,
    tool,
    action: "probe",
    projectRoot: ctx.projectRoot,
    durationMs: Date.now() - startedAt,
    ok: false,
    errorCategory: error.category,
  });
  return err(error);
}

export function passProbe(
  ctx: ToolContext,
  tool: string,
  probe: ProbeName,
  attempt: 1 | 2,
  startedAt: number,
): ToolResult<ProbeAcknowledgement> {
  ctx.guard.after({
    sessionId: ctx.sessionId,
    agent: ctx.agent,
    tool,
    action: "probe",
    projectRoot: ctx.projectRoot,
    durationMs: Date.now() - startedAt,
    ok: true,
  });
  return ok({
    acknowledgement: "PASS",
    codeaProbe: probe,
    codeaProbeResult: "pass",
    codeaProbeAttempt: String(attempt) as "1" | "2",
  });
}
