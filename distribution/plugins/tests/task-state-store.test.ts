import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { TaskStateError, TaskStateStore } from "../src/task-state/store";

function tempDir(prefix: string): string {
  return fs.mkdtempSync(path.join(os.tmpdir(), prefix));
}

function steps(count: number) {
  return Array.from({ length: count }, (_, i) => ({
    id: `step-${i + 1}`,
    title: `Step ${i + 1}`,
    verification: `verify ${i + 1}`,
  }));
}

function makeStore(root?: string, home?: string) {
  const workspaceRoot = root ?? tempDir("codea task29 workspace 中文 ");
  const codeaHome = home ?? tempDir("codea task29 home 中文 ");
  return {
    workspaceRoot,
    codeaHome,
    store: new TaskStateStore({ workspaceRoot, codeaHome }),
  };
}

function onlyStateFile(codeaHome: string): string {
  const taskStateRoot = path.join(codeaHome, "task-state");
  const workspaceDirs = fs.readdirSync(taskStateRoot);
  expect(workspaceDirs).toHaveLength(1);
  const workspaceDir = path.join(taskStateRoot, workspaceDirs[0]!);
  const files = fs.readdirSync(workspaceDir);
  expect(files.filter((f) => f.endsWith(".json"))).toHaveLength(1);
  expect(files.some((f) => f.includes(".tmp"))).toBe(false);
  return path.join(workspaceDir, files.find((f) => f.endsWith(".json"))!);
}

async function expectTaskStateError(fn: () => Promise<unknown>, code?: string) {
  try {
    await fn();
    throw new Error("expected TaskStateError");
  } catch (error) {
    expect(error).toBeInstanceOf(TaskStateError);
    if (code) expect((error as TaskStateError).code).toBe(code);
  }
}

describe("TaskStateStore invariants", () => {
  test("accepts exactly 3 through 7 steps", async () => {
    for (const count of [3, 4, 5, 6, 7]) {
      const { store } = makeStore();
      const plan = await store.create(`session-${count}`, "Implement bounded planning", steps(count));
      expect(plan.steps).toHaveLength(count);
      expect(plan.steps.every((step) => step.status === "pending")).toBe(true);
      expect(plan.version).toBe(1);
    }
  });

  test("rejects plans with 2 or 8 steps", async () => {
    for (const count of [2, 8]) {
      const { store } = makeStore();
      await expectTaskStateError(() => store.create("session", "goal", steps(count)), "TASK_STATE_INVALID");
    }
  });

  test("rejects duplicate step ids and blank metadata", async () => {
    const { store } = makeStore();
    await expectTaskStateError(
      () => store.create("session", "goal", [
        { id: "same", title: "A" },
        { id: "same", title: "B" },
        { id: "third", title: "C" },
      ]),
      "TASK_STATE_INVALID",
    );
    await expectTaskStateError(() => store.create("session", "   ", steps(3)), "TASK_STATE_INVALID");
    await expectTaskStateError(
      () => store.create("session", "goal", [
        { id: "a", title: "A" },
        { id: "b", title: "  " },
        { id: "c", title: "C" },
      ]),
      "TASK_STATE_INVALID",
    );
  });

  test("enforces metadata length limits", async () => {
    const { store } = makeStore();
    await expectTaskStateError(() => store.create("s", "g".repeat(1001), steps(3)), "TASK_STATE_INVALID");
    await expectTaskStateError(
      () => store.create("s", "goal", [
        { id: "a", title: "t".repeat(301) },
        { id: "b", title: "B" },
        { id: "c", title: "C" },
      ]),
      "TASK_STATE_INVALID",
    );
    await expectTaskStateError(
      () => store.create("s", "goal", [
        { id: "a", title: "A", verification: "v".repeat(501) },
        { id: "b", title: "B" },
        { id: "c", title: "C" },
      ]),
      "TASK_STATE_INVALID",
    );
  });

  test("permits only one in_progress step and legal transitions", async () => {
    const { store } = makeStore();
    await store.create("s", "goal", steps(3));
    await store.updateStep("s", "step-1", "in_progress");
    await expectTaskStateError(() => store.updateStep("s", "step-2", "in_progress"), "TASK_STATE_INVALID");
    await expectTaskStateError(() => store.updateStep("s", "step-2", "completed", "not started"), "TASK_STATE_INVALID");
    await store.updateStep("s", "step-1", "completed", "tests pass");
    await expectTaskStateError(() => store.updateStep("s", "step-1", "pending" as any), "TASK_STATE_INVALID");
    await expectTaskStateError(() => store.updateStep("s", "step-1", "in_progress"), "TASK_STATE_INVALID");
  });

  test("blocked requires concise evidence", async () => {
    const { store } = makeStore();
    await store.create("s", "goal", steps(3));
    await store.updateStep("s", "step-1", "in_progress");
    await expectTaskStateError(() => store.updateStep("s", "step-1", "blocked"), "TASK_STATE_INVALID");
    await expectTaskStateError(
      () => store.updateStep("s", "step-1", "blocked", "e".repeat(1001)),
      "TASK_STATE_INVALID",
    );
    const plan = await store.updateStep("s", "step-1", "blocked", "dependency unavailable");
    expect(plan.steps[0]!.status).toBe("blocked");
    expect(plan.steps[0]!.evidence).toBe("dependency unavailable");
  });

  test("new plan replaces only the same session", async () => {
    const { store } = makeStore();
    await store.create("session-a", "A1", steps(3));
    await store.create("session-b", "B1", steps(3));
    await store.create("session-a", "A2", steps(4));
    expect((await store.load("session-a"))?.goal).toBe("A2");
    expect((await store.load("session-b"))?.goal).toBe("B1");
  });
});

describe("TaskStateStore persistence", () => {
  test("persists atomically under hashed workspace and session names", async () => {
    const { store, workspaceRoot, codeaHome } = makeStore();
    const sessionId = "runtime/session raw 中文";
    await store.create(sessionId, "goal", steps(3));

    const taskStateRoot = path.join(codeaHome, "task-state");
    const workspaceDirs = fs.readdirSync(taskStateRoot);
    expect(workspaceDirs).toHaveLength(1);
    expect(workspaceDirs[0]).toMatch(/^[a-f0-9]{64}$/);
    expect(workspaceDirs[0]).not.toContain(path.basename(workspaceRoot));

    const stateFile = onlyStateFile(codeaHome);
    expect(path.basename(stateFile)).toMatch(/^[a-f0-9]{64}\.json$/);
    expect(path.basename(stateFile)).not.toContain("session raw");
    expect(JSON.parse(fs.readFileSync(stateFile, "utf8")).goal).toBe("goal");
  });

  test("reloads the same session after store recreation", async () => {
    const { store, workspaceRoot, codeaHome } = makeStore();
    await store.create("session-a", "restart safe", steps(3));
    await store.updateStep("session-a", "step-1", "in_progress");
    const recreated = new TaskStateStore({ workspaceRoot, codeaHome });
    const plan = await recreated.load("session-a");
    expect(plan?.goal).toBe("restart safe");
    expect(plan?.steps[0]?.status).toBe("in_progress");
  });

  test("malformed or unknown persisted fields fail closed", async () => {
    const { store, codeaHome } = makeStore();
    await store.create("session-a", "goal", steps(3));
    const stateFile = onlyStateFile(codeaHome);

    fs.writeFileSync(stateFile, "{not-json", "utf8");
    await expectTaskStateError(() => store.load("session-a"), "TASK_STATE_CORRUPT");

    await store.create("session-a", "goal", steps(3));
    const parsed = JSON.parse(fs.readFileSync(stateFile, "utf8"));
    parsed.rawSource = "class Secret {}";
    fs.writeFileSync(stateFile, JSON.stringify(parsed), "utf8");
    await expectTaskStateError(() => store.load("session-a"), "TASK_STATE_CORRUPT");
  });

  test("spaces and non-ASCII CODEA_HOME paths work", async () => {
    const root = tempDir("workspace 中文 spaces ");
    const home = tempDir("CODEA HOME 中文 spaces ");
    const store = new TaskStateStore({ workspaceRoot: root, codeaHome: home });
    await store.create("会话 session", "目标 goal", steps(3));
    expect((await store.load("会话 session"))?.goal).toBe("目标 goal");
  });

  test("clear removes only that session and hasActionablePlan reflects remaining work", async () => {
    const { store } = makeStore();
    await store.create("a", "goal", steps(3));
    await store.create("b", "goal-b", steps(3));
    expect(await store.hasActionablePlan("a")).toBe(true);
    await store.clear("a");
    expect(await store.load("a")).toBeNull();
    expect((await store.load("b"))?.goal).toBe("goal-b");
  });
});
