import type { TaskStateStore } from "../task-state/store";
import type { StepStatus } from "../task-state/types";
import { ToolError } from "./errors";
import { planningToolError, summarizeTaskPlan, type TaskPlanSummary } from "./task-plan";
import { err, ok, type ToolContext, type ToolResult } from "./types";

export interface TaskStepInput {
  stepId: string;
  status: Exclude<StepStatus, "pending">;
  evidence?: string;
}

export function createTaskStepTool(store: TaskStateStore) {
  return {
    name: "task_step",
    description: "Move exactly one plan step with concise evidence.",

    async execute(params: unknown, ctx: ToolContext): Promise<ToolResult<{ plan: TaskPlanSummary }>> {
      const started = Date.now();
      try {
        const input = params as TaskStepInput;
        if (!input || typeof input.stepId !== "string" || !["in_progress", "completed", "blocked"].includes(input.status)) {
          throw new ToolError("INVALID_INPUT", "task_step requires stepId and a supported status");
        }
        if ((input.status === "completed" || input.status === "blocked") && !input.evidence?.trim()) {
          throw new ToolError("TASK_STATE_INVALID", `${input.status} requires concise evidence`);
        }
        if (!ctx.rootTurnId?.trim()) throw new ToolError("PLAN_REQUIRED", "task_step requires a current root turn");
        const plan = await store.updateStep(ctx.sessionId, input.stepId, input.status, input.evidence, ctx.rootTurnId);
        ctx.guard.after({
          sessionId: ctx.sessionId, agent: ctx.agent, tool: "task_step", action: "plan",
          projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: true,
        });
        return ok({ plan: summarizeTaskPlan(plan) });
      } catch (error) {
        const toolErr = planningToolError(error, "task_step failed");
        ctx.guard.after({
          sessionId: ctx.sessionId, agent: ctx.agent, tool: "task_step", action: "plan",
          projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: false,
          errorCategory: toolErr.category,
        });
        return err(toolErr);
      }
    },
  };
}
