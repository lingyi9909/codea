import { beginProbe, exactObject, passProbe, ProbeAttemptState, rejectProbe, type ProbeAcknowledgement } from "./probe-state";
import type { ToolContext, ToolResult } from "./types";

const TOOL = "probe_structured";

function validStructured(params: unknown): boolean {
  if (!exactObject(params, ["items", "summary"])) return false;
  if (!Array.isArray(params.items) || params.items.length !== 2) return false;
  const first = params.items[0];
  const second = params.items[1];
  if (!exactObject(first, ["id", "enabled"]) || first.id !== "A" || first.enabled !== true) return false;
  if (!exactObject(second, ["id", "enabled"]) || second.id !== "B" || second.enabled !== false) return false;
  if (!exactObject(params.summary, ["count"]) || params.summary.count !== 2) return false;
  return true;
}

export function createProbeStructuredTool(state: ProbeAttemptState) {
  return {
    name: TOOL,
    description: "Deterministic non-mutating structured-output qualification probe.",
    async execute(params: unknown, ctx: ToolContext): Promise<ToolResult<ProbeAcknowledgement>> {
      const startedAt = Date.now();
      const begun = beginProbe(state, ctx, TOOL, "structured", startedAt);
      if ("limited" in begun) return begun.limited;
      if (!validStructured(params)) {
        return rejectProbe(ctx, TOOL, "probe_structured requires the exact qualification payload", startedAt);
      }
      return passProbe(ctx, TOOL, "structured", begun.attempt, startedAt);
    },
  };
}
