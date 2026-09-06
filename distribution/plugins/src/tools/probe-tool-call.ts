import { beginProbe, exactObject, passProbe, ProbeAttemptState, rejectProbe, type ProbeAcknowledgement } from "./probe-state";
import type { ToolContext, ToolResult } from "./types";

const TOOL = "probe_tool_call";

export function createProbeToolCallTool(state: ProbeAttemptState) {
  return {
    name: TOOL,
    description: "Deterministic non-mutating tool-call qualification probe.",
    async execute(params: unknown, ctx: ToolContext): Promise<ToolResult<ProbeAcknowledgement>> {
      const startedAt = Date.now();
      const begun = beginProbe(state, ctx, TOOL, "tool_call", startedAt);
      if ("limited" in begun) return begun.limited;
      if (!exactObject(params, ["token", "value"]) || params.token !== "CODEA-17" || params.value !== 42) {
        return rejectProbe(ctx, TOOL, "probe_tool_call requires the exact qualification payload", startedAt);
      }
      return passProbe(ctx, TOOL, "tool_call", begun.attempt, startedAt);
    },
  };
}
