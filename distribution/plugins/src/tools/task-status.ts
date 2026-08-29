import type { TaskStateStore } from "../task-state/store";
import { planningToolError, summarizeTaskPlan, type TaskPlanSummary } from "./task-plan";
import { err, ok, type ToolContext, type ToolResult } from "./types";

export function createTaskStatusTool(store: TaskStateStore) {
  return {
    name: "task_status",
    description: "Reread current machine task state.",

    async execute(_params: unknown, ctx: ToolContext): Promise<ToolResult<{ plan: TaskPlanSummary | null }>> {
      const started = Date.now();
      try {
        if (!ctx.rootTurnId?.trim()) {
          ctx.guard.after({
            sessionId: ctx.sessionId, agent: ctx.agent, tool: "task_status", action: "plan",
            projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: true,
          });
          return ok({ plan: null });
        }
        const plan = await store.loadForRoot(ctx.sessionId, ctx.rootTurnId);
        ctx.guard.after({
          sessionId: ctx.sessionId, agent: ctx.agent, tool: "task_status", action: "plan",
          projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: true,
        });
        return ok({ plan: plan ? summarizeTaskPlan(plan) : null });
      } catch (error) {
        const toolErr = planningToolError(error, "task_status failed");
        ctx.guard.after({
          sessionId: ctx.sessionId, agent: ctx.agent, tool: "task_status", action: "plan",
          projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: false,
          errorCategory: toolErr.category,
        });
        return err(toolErr);
      }
    },
  };
}
