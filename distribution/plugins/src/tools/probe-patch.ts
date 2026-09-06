import { beginProbe, exactObject, passProbe, ProbeAttemptState, rejectProbe, type ProbeAcknowledgement } from "./probe-state";
import type { ToolContext, ToolResult } from "./types";

const TOOL = "probe_patch";

function validPatch(params: unknown): boolean {
  return exactObject(params, ["original", "instruction", "result"])
    && params.original === "alpha\nbeta\n"
    && params.instruction === "replace beta with gamma"
    && params.result === "alpha\ngamma\n";
}

export function createProbePatchTool(state: ProbeAttemptState) {
  return {
    name: TOOL,
    description: "Deterministic non-mutating patch-following qualification probe.",
    async execute(params: unknown, ctx: ToolContext): Promise<ToolResult<ProbeAcknowledgement>> {
      const startedAt = Date.now();
      const begun = beginProbe(state, ctx, TOOL, "patch", startedAt);
      if ("limited" in begun) return begun.limited;
      if (!validPatch(params)) {
        return rejectProbe(ctx, TOOL, "probe_patch requires the exact qualification payload", startedAt);
      }
      return passProbe(ctx, TOOL, "patch", begun.attempt, startedAt);
    },
  };
}
