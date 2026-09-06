import { beginProbe, exactObject, passProbe, ProbeAttemptState, rejectProbe, type ProbeAcknowledgement } from "./probe-state";
import type { ToolContext, ToolResult } from "./types";

const TOOL = "probe_plan";

function validPlan(params: unknown): boolean {
  if (!exactObject(params, ["steps"]) || !Array.isArray(params.steps) || params.steps.length !== 3) return false;
  const expected = [
    ["1", "inspect"],
    ["2", "edit"],
    ["3", "verify"],
  ] as const;
  return params.steps.every((step, index) => {
    const want = expected[index];
    return !!want && exactObject(step, ["id", "action"]) && step.id === want[0] && step.action === want[1];
  });
}

export function createProbePlanTool(state: ProbeAttemptState) {
  return {
    name: TOOL,
    description: "Deterministic non-mutating planning qualification probe.",
    async execute(params: unknown, ctx: ToolContext): Promise<ToolResult<ProbeAcknowledgement>> {
      const startedAt = Date.now();
      const begun = beginProbe(state, ctx, TOOL, "planning", startedAt);
      if ("limited" in begun) return begun.limited;
      if (!validPlan(params)) {
        return rejectProbe(ctx, TOOL, "probe_plan requires the exact qualification payload", startedAt);
      }
      return passProbe(ctx, TOOL, "planning", begun.attempt, startedAt);
    },
  };
}
