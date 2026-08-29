import { TaskStateError, type TaskStateStore } from "../task-state/store";
import type { NewStep, TaskPlan } from "../task-state/types";
import { ToolError } from "./errors";
import { err, ok, type ToolContext, type ToolResult } from "./types";

export interface TaskPlanInput {
  goal: string;
  steps: NewStep[];
}

export interface TaskPlanSummary {
  goal: string;
  steps: Array<{
    id: string;
    title: string;
    verification?: string;
    status: string;
    evidence?: string;
  }>;
}

export function summarizeTaskPlan(plan: TaskPlan): TaskPlanSummary {
  return {
    goal: plan.goal,
    steps: plan.steps.map((step) => ({
      id: step.id,
      title: step.title,
      ...(step.verification === undefined ? {} : { verification: step.verification }),
      status: step.status,
      ...(step.evidence === undefined ? {} : { evidence: step.evidence }),
    })),
  };
}

export function planningToolError(error: unknown, fallback: string): ToolError {
  if (error instanceof ToolError) return error;
  if (error instanceof TaskStateError) return new ToolError(error.code, error.message, error);
  return new ToolError("INTERNAL_ERROR", error instanceof Error ? error.message || fallback : fallback, error);
}

export function createTaskPlanTool(store: TaskStateStore) {
  return {
    name: "task_plan",
    description: "Establish bounded execution state before project mutation/command execution.",

    async execute(params: unknown, ctx: ToolContext): Promise<ToolResult<{ plan: TaskPlanSummary }>> {
      const started = Date.now();
      try {
        const input = params as TaskPlanInput;
        if (!input || typeof input.goal !== "string" || !Array.isArray(input.steps)) {
          throw new ToolError("INVALID_INPUT", "task_plan requires goal and steps");
        }
        const plan = await store.create(ctx.sessionId, input.goal, input.steps);
        ctx.guard.after({
          sessionId: ctx.sessionId, agent: ctx.agent, tool: "task_plan", action: "plan",
          projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: true,
        });
        return ok({ plan: summarizeTaskPlan(plan) });
      } catch (error) {
        const toolErr = planningToolError(error, "task_plan failed");
        ctx.guard.after({
          sessionId: ctx.sessionId, agent: ctx.agent, tool: "task_plan", action: "plan",
          projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: false,
          errorCategory: toolErr.category,
        });
        return err(toolErr);
      }
    },
  };
}
