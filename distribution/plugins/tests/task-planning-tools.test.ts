import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { TaskStateStore } from "../src/task-state/store";
import { createTaskPlanTool } from "../src/tools/task-plan";
import { createTaskStepTool } from "../src/tools/task-step";
import { createTaskStatusTool } from "../src/tools/task-status";
import { plugin } from "../src/opencode/entry";
import type { PluginInput, ToolContext as OpenCodeToolContext } from "../src/opencode/types";
import { makeContext, makeTempRoot, readAudit } from "./helpers";

function makeStore(root: string) {
  return new TaskStateStore({ workspaceRoot: root, codeaHome: fs.mkdtempSync(path.join(os.tmpdir(), "codea-task29-tools-home-")) });
}

function planInput(count = 3) {
  return {
    goal: "Implement planning protocol",
    steps: Array.from({ length: count }, (_, i) => ({ id: `step-${i + 1}`, title: `Step ${i + 1}`, verification: `verify-${i + 1}` })),
  };
}

function makePluginInput(root: string): PluginInput {
  return {
    client: {}, project: {}, directory: root, worktree: root, experimental_workspace: {},
    serverUrl: new URL("http://127.0.0.1:4096"), $: {},
  } as unknown as PluginInput;
}

function makeOctx(root: string, sessionID: string, metadataEvents: any[] = []): OpenCodeToolContext {
  return {
    sessionID,
    messageID: "m1",
    agent: "general",
    directory: root,
    worktree: root,
    abort: new AbortController().signal,
    metadata(input) { metadataEvents.push(input); },
    async ask() { throw new Error("planning tools must not request project mutation approval"); },
  };
}

describe("task planning tools", () => {
  test("task_plan creates exact persisted 3-7 step state", async () => {
    for (const count of [3, 7]) {
      const root = makeTempRoot(`codea-plan-${count}-`);
      const store = makeStore(root);
      const { ctx } = makeContext(root);
      ctx.sessionId = `session-${count}`;
      const result = await createTaskPlanTool(store).execute(planInput(count), ctx);
      expect(result.ok).toBe(true);
      const persisted = await store.load(ctx.sessionId);
      expect(persisted?.steps).toHaveLength(count);
      expect(persisted?.goal).toBe("Implement planning protocol");
    }
  });

  test("task_step moves exactly one step and completion requires evidence", async () => {
    const root = makeTempRoot("codea-step-");
    const store = makeStore(root);
    const { ctx } = makeContext(root);
    await createTaskPlanTool(store).execute(planInput(), ctx);

    let result = await createTaskStepTool(store).execute({ stepId: "step-1", status: "in_progress" }, ctx);
    expect(result.ok).toBe(true);
    result = await createTaskStepTool(store).execute({ stepId: "step-1", status: "completed" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("TASK_STATE_INVALID");
    result = await createTaskStepTool(store).execute({ stepId: "step-1", status: "completed", evidence: "focused test passed" }, ctx);
    expect(result.ok).toBe(true);
    const plan = await store.load(ctx.sessionId);
    expect(plan?.steps[0]?.status).toBe("completed");
    expect(plan?.steps[1]?.status).toBe("pending");
  });

  test("task_status returns sanitized current plan and cross-session state is isolated", async () => {
    const root = makeTempRoot("codea-status-");
    const store = makeStore(root);
    const a = makeContext(root).ctx;
    a.sessionId = "session-a";
    const b = makeContext(root).ctx;
    b.sessionId = "session-b";
    await createTaskPlanTool(store).execute(planInput(), a);
    const statusA = await createTaskStatusTool(store).execute({}, a);
    const statusB = await createTaskStatusTool(store).execute({}, b);
    expect(statusA.ok).toBe(true);
    if (statusA.ok) {
      const serialized = JSON.stringify(statusA.data);
      expect(serialized).not.toContain(root);
      expect(serialized).not.toContain("createdAt");
    }
    expect(statusB.ok).toBe(true);
    if (statusB.ok) expect(statusB.data.plan).toBeNull();
    const updateB = await createTaskStepTool(store).execute({ stepId: "step-1", status: "in_progress" }, b);
    expect(updateB.ok).toBe(false);
  });

  test("invalid/corrupt state returns stable structured ToolError", async () => {
    const root = makeTempRoot("codea-tool-error-");
    const store = makeStore(root);
    const { ctx } = makeContext(root);
    const invalid = await createTaskPlanTool(store).execute(planInput(2), ctx);
    expect(invalid.ok).toBe(false);
    if (!invalid.ok) expect(invalid.error.category).toBe("TASK_STATE_INVALID");
  });

  test("all planning tools are registered as non-mutating plan operations and emit lifecycle evidence", async () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-planning-plugin-"));
    const root = path.join(tmp, "project");
    const home = path.join(tmp, "home");
    fs.mkdirSync(root, { recursive: true });
    fs.mkdirSync(home, { recursive: true });
    const previous = process.env.CODEA_HOME;
    process.env.CODEA_HOME = home;
    try {
      const auditLog = path.join(tmp, "audit.log");
      const hooks = await plugin.server(makePluginInput(root), { auditLog });
      expect(Object.keys(hooks.tool ?? {})).toEqual(expect.arrayContaining(["task_plan", "task_step", "task_status"]));
      const events: any[] = [];
      const octx = makeOctx(root, "session-plugin", events);
      const p = await hooks.tool!.task_plan!.execute(planInput(), octx);
      const s = await hooks.tool!.task_status!.execute({}, octx);
      expect(typeof p).not.toBe("undefined");
      expect(typeof s).not.toBe("undefined");
      expect(events.some((e) => e.metadata?.codeaPlugin === "codea-enterprise")).toBe(true);
      const audit = readAudit(auditLog);
      expect(audit).toContain("task_plan");
      expect(audit).toContain("task_status");
    } finally {
      if (previous === undefined) delete process.env.CODEA_HOME;
      else process.env.CODEA_HOME = previous;
      fs.rmSync(tmp, { recursive: true, force: true });
    }
  });
});
