import { createHash, randomBytes, randomUUID } from "node:crypto";
import * as fs from "node:fs/promises";
import * as path from "node:path";
import { z } from "zod";
import { TASK_PLAN_LIMITS, type NewStep, type StepStatus, type TaskPlan } from "./types";

export type TaskStateErrorCode = "TASK_STATE_INVALID" | "TASK_STATE_CORRUPT";

export class TaskStateError extends Error {
  readonly code: TaskStateErrorCode;

  constructor(code: TaskStateErrorCode, message: string, cause?: unknown) {
    super(message, cause === undefined ? undefined : { cause });
    this.name = "TaskStateError";
    this.code = code;
  }
}

const stepSchema = z.object({
  id: z.string().min(1).max(TASK_PLAN_LIMITS.stepIDChars),
  title: z.string().min(1).max(TASK_PLAN_LIMITS.stepTitleChars),
  verification: z.string().max(TASK_PLAN_LIMITS.verificationChars).optional(),
  status: z.enum(["pending", "in_progress", "completed", "blocked"]),
  evidence: z.string().max(TASK_PLAN_LIMITS.evidenceChars).optional(),
}).strict();

const planSchema = z.object({
  id: z.string().uuid(),
  sessionId: z.string().min(1),
  rootMessageID: z.string().min(1),
  taskEpoch: z.string().min(1),
  goal: z.string().min(1).max(TASK_PLAN_LIMITS.goalChars),
  steps: z.array(stepSchema).min(TASK_PLAN_LIMITS.minSteps).max(TASK_PLAN_LIMITS.maxSteps),
  createdAt: z.string().datetime({ offset: true }),
  updatedAt: z.string().datetime({ offset: true }),
  version: z.literal(2),
}).strict();

export interface TaskStateStoreOptions {
  workspaceRoot: string;
  codeaHome?: string;
}

function sha256(value: string): string {
  return createHash("sha256").update(value, "utf8").digest("hex");
}

function normalizeWorkspaceRoot(root: string): string {
  const normalized = path.resolve(root).replace(/\\/g, "/").replace(/\/$/, "");
  return process.platform === "win32" ? normalized.toLowerCase() : normalized;
}

function validateText(value: string, label: string, max: number): string {
  const trimmed = value.trim();
  if (!trimmed) throw new TaskStateError("TASK_STATE_INVALID", `${label} must not be blank`);
  if (value.length > max) throw new TaskStateError("TASK_STATE_INVALID", `${label} exceeds ${max} characters`);
  return value;
}

function validateNewSteps(steps: NewStep[]): void {
  if (steps.length < TASK_PLAN_LIMITS.minSteps || steps.length > TASK_PLAN_LIMITS.maxSteps) {
    throw new TaskStateError(
      "TASK_STATE_INVALID",
      `task plan must contain ${TASK_PLAN_LIMITS.minSteps}-${TASK_PLAN_LIMITS.maxSteps} steps`,
    );
  }
  const ids = new Set<string>();
  for (const step of steps) {
    validateText(step.id, "step id", TASK_PLAN_LIMITS.stepIDChars);
    validateText(step.title, "step title", TASK_PLAN_LIMITS.stepTitleChars);
    if (step.verification !== undefined && step.verification.length > TASK_PLAN_LIMITS.verificationChars) {
      throw new TaskStateError(
        "TASK_STATE_INVALID",
        `step verification exceeds ${TASK_PLAN_LIMITS.verificationChars} characters`,
      );
    }
    if (ids.has(step.id)) throw new TaskStateError("TASK_STATE_INVALID", `duplicate step id: ${step.id}`);
    ids.add(step.id);
  }
}

function assertPlanInvariants(plan: TaskPlan): void {
  if (plan.rootMessageID !== plan.taskEpoch) {
    throw new TaskStateError("TASK_STATE_CORRUPT", "task plan epoch does not match root message identity");
  }
  const active = plan.steps.filter((step) => step.status === "in_progress");
  if (active.length > 1) throw new TaskStateError("TASK_STATE_CORRUPT", "more than one in_progress step");
  for (const step of plan.steps) {
    if (step.status === "blocked" && !step.evidence?.trim()) {
      throw new TaskStateError("TASK_STATE_CORRUPT", `blocked step ${step.id} is missing evidence`);
    }
  }
  const ids = new Set(plan.steps.map((step) => step.id));
  if (ids.size !== plan.steps.length) throw new TaskStateError("TASK_STATE_CORRUPT", "duplicate step ids");
}

export class TaskStateStore {
  private readonly taskStateRoot: string;

  constructor(options: TaskStateStoreOptions) {
    if (!options.workspaceRoot?.trim()) {
      throw new TaskStateError("TASK_STATE_INVALID", "workspaceRoot is required");
    }
    const codeaHome = options.codeaHome ?? process.env.CODEA_HOME;
    if (!codeaHome?.trim()) {
      throw new TaskStateError("TASK_STATE_INVALID", "CODEA_HOME is required");
    }
    const workspaceHash = sha256(normalizeWorkspaceRoot(options.workspaceRoot));
    this.taskStateRoot = path.join(codeaHome, "task-state", workspaceHash);
  }

  async load(sessionId: string): Promise<TaskPlan | null> {
    const file = this.sessionFile(sessionId);
    let raw: string;
    try {
      raw = await fs.readFile(file, "utf8");
    } catch (error: any) {
      if (error?.code === "ENOENT") return null;
      throw new TaskStateError("TASK_STATE_CORRUPT", "unable to read persisted task state", error);
    }

    try {
      const parsed = JSON.parse(raw);
      const plan = planSchema.parse(parsed) as TaskPlan;
      if (plan.sessionId !== sessionId) {
        throw new TaskStateError("TASK_STATE_CORRUPT", "persisted task state session does not match file identity");
      }
      assertPlanInvariants(plan);
      return plan;
    } catch (error) {
      if (error instanceof TaskStateError) throw error;
      throw new TaskStateError("TASK_STATE_CORRUPT", "persisted task state is malformed", error);
    }
  }

  async create(sessionId: string, goal: string, steps: NewStep[], rootMessageID = sessionId): Promise<TaskPlan> {
    validateText(sessionId, "session id", 1000);
    validateText(rootMessageID, "root message id", 1000);
    validateText(goal, "goal", TASK_PLAN_LIMITS.goalChars);
    validateNewSteps(steps);

    const now = new Date().toISOString();
    const plan: TaskPlan = {
      id: randomUUID(),
      sessionId,
      rootMessageID,
      taskEpoch: rootMessageID,
      goal,
      steps: steps.map((step) => ({
        id: step.id,
        title: step.title,
        ...(step.verification === undefined ? {} : { verification: step.verification }),
        status: "pending",
      })),
      createdAt: now,
      updatedAt: now,
      version: 2,
    };
    await this.persist(plan);
    return plan;
  }

  async updateStep(
    sessionId: string,
    stepId: string,
    status: StepStatus,
    evidence?: string,
    rootMessageID?: string,
  ): Promise<TaskPlan> {
    const plan = await this.load(sessionId);
    if (!plan) throw new TaskStateError("TASK_STATE_INVALID", "task plan does not exist for session");
    if (rootMessageID !== undefined && plan.rootMessageID !== rootMessageID) {
      throw new TaskStateError("TASK_STATE_INVALID", "task plan root does not match current root turn");
    }
    const step = plan.steps.find((candidate) => candidate.id === stepId);
    if (!step) throw new TaskStateError("TASK_STATE_INVALID", `unknown step id: ${stepId}`);
    if (evidence !== undefined && evidence.length > TASK_PLAN_LIMITS.evidenceChars) {
      throw new TaskStateError(
        "TASK_STATE_INVALID",
        `step evidence exceeds ${TASK_PLAN_LIMITS.evidenceChars} characters`,
      );
    }

    const from = step.status;
    const allowed =
      (from === "pending" && status === "in_progress") ||
      (from === "in_progress" && (status === "completed" || status === "blocked")) ||
      (from === "blocked" && status === "in_progress");
    if (!allowed) {
      throw new TaskStateError("TASK_STATE_INVALID", `illegal step transition: ${from} -> ${status}`);
    }
    if (status === "in_progress") {
      const otherActive = plan.steps.some((candidate) => candidate.id !== stepId && candidate.status === "in_progress");
      if (otherActive) throw new TaskStateError("TASK_STATE_INVALID", "only one step may be in_progress");
    }
    if (status === "blocked" && !evidence?.trim()) {
      throw new TaskStateError("TASK_STATE_INVALID", "blocked step requires evidence");
    }

    step.status = status;
    if (evidence === undefined || evidence === "") delete step.evidence;
    else step.evidence = evidence;
    plan.updatedAt = new Date().toISOString();
    await this.persist(plan);
    return plan;
  }

  async clear(sessionId: string): Promise<void> {
    try {
      await fs.unlink(this.sessionFile(sessionId));
    } catch (error: any) {
      if (error?.code !== "ENOENT") throw error;
    }
  }

  async loadForRoot(sessionId: string, rootMessageID: string): Promise<TaskPlan | null> {
    if (!rootMessageID.trim()) return null;
    const plan = await this.load(sessionId);
    if (!plan || plan.rootMessageID !== rootMessageID || plan.taskEpoch !== rootMessageID) return null;
    return plan;
  }

  async hasActionablePlan(sessionId: string, rootMessageID: string): Promise<boolean> {
    const plan = await this.loadForRoot(sessionId, rootMessageID);
    if (!plan) return false;
    return plan.steps.some((step) => step.status === "pending" || step.status === "in_progress");
  }

  private sessionFile(sessionId: string): string {
    validateText(sessionId, "session id", 1000);
    return path.join(this.taskStateRoot, `${sha256(sessionId)}.json`);
  }

  private async persist(plan: TaskPlan): Promise<void> {
    const parsed = planSchema.safeParse(plan);
    if (!parsed.success) {
      throw new TaskStateError("TASK_STATE_INVALID", "task plan failed schema validation", parsed.error);
    }
    assertPlanInvariants(plan);
    await fs.mkdir(this.taskStateRoot, { recursive: true, mode: 0o700 });
    const file = this.sessionFile(plan.sessionId);
    const temp = `${file}.${process.pid}.${randomBytes(8).toString("hex")}.tmp`;
    try {
      await fs.writeFile(temp, `${JSON.stringify(plan)}\n`, { encoding: "utf8", mode: 0o600 });
      await fs.rename(temp, file);
    } catch (error) {
      throw new TaskStateError("TASK_STATE_CORRUPT", "unable to persist task state atomically", error);
    } finally {
      await fs.rm(temp, { force: true }).catch(() => {});
    }
  }
}
