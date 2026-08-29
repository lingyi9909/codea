import type { TaskStateStore } from "./store";

export class PlanRequiredError extends Error {
  readonly code = "PLAN_REQUIRED" as const;
  readonly category = "PLAN_REQUIRED" as const;
  readonly operation: string;

  constructor(operation: string, cause?: unknown) {
    super(`PLAN_REQUIRED: create or restore an actionable task plan before ${operation}`, cause === undefined ? undefined : { cause });
    this.name = "PlanRequiredError";
    this.operation = operation;
  }
}

export async function requirePlan(store: TaskStateStore, sessionId: string, operation: string): Promise<void> {
  try {
    if (await store.hasActionablePlan(sessionId)) return;
  } catch (error) {
    throw new PlanRequiredError(operation, error);
  }
  throw new PlanRequiredError(operation);
}
