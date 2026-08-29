import { describe, expect, test } from "bun:test";
import * as crypto from "node:crypto";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { plugin } from "../src/opencode/entry";
import type { PluginInput, ToolContext } from "../src/opencode/types";

function input(root: string): PluginInput {
  return { client: {}, project: {}, directory: root, worktree: root, experimental_workspace: {}, serverUrl: new URL("http://127.0.0.1:4096"), $: {} } as unknown as PluginInput;
}

function context(root: string, sessionID: string): ToolContext {
  return {
    sessionID, messageID: `turn-${sessionID}`, agent: "general", directory: root, worktree: root,
    abort: new AbortController().signal, metadata() {}, async ask() {},
  };
}

const plan = {
  goal: "Prove isolated execution",
  steps: [
    { id: "inspect", title: "Inspect" },
    { id: "change", title: "Change" },
    { id: "verify", title: "Verify" },
  ],
};

function normalizeWorkspace(root: string): string {
  const normalized = path.resolve(root).replace(/\\/g, "/").replace(/\/$/, "");
  return process.platform === "win32" ? normalized.toLowerCase() : normalized;
}

function stateFile(home: string, root: string, sessionID: string): string {
  const hash = (value: string) => crypto.createHash("sha256").update(value, "utf8").digest("hex");
  return path.join(home, "task-state", hash(normalizeWorkspace(root)), `${hash(sessionID)}.json`);
}

describe("Task 29 planning isolation and recovery", () => {
  test("session plans are isolated and persist across plugin restart", async () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-task29-isolation-"));
    const root = path.join(tmp, "project");
    const home = path.join(tmp, "home");
    fs.mkdirSync(root, { recursive: true });
    fs.mkdirSync(home, { recursive: true });
    const previous = process.env.CODEA_HOME;
    process.env.CODEA_HOME = home;
    try {
      const first = await plugin.server(input(root), { auditLog: path.join(tmp, "audit-1.log") });
      await first.tool!.task_plan!.execute(plan, context(root, "session-A"));

      await expect(first["tool.execute.before"]!({ tool: "write", sessionID: "session-B", callID: "b1" }, { args: { filePath: "safe.txt", content: "x" } })).rejects.toThrow(/PLAN_REQUIRED/);
      await expect(first["tool.execute.before"]!({ tool: "write", sessionID: "session-A", callID: "a1" }, { args: { filePath: "..\/outside.txt", content: "x" } })).rejects.toThrow(/native-path:outside-project/);

      const restarted = await plugin.server(input(root), { auditLog: path.join(tmp, "audit-2.log") });
      const status = await restarted.tool!.task_status!.execute({}, context(root, "session-A"));
      expect(status.metadata?.codeaTaskPlan).toBe("true");
      expect(status.metadata?.codeaPlanTotal).toBe("3");
      await expect(restarted["tool.execute.before"]!({ tool: "write", sessionID: "session-A", callID: "a2" }, { args: { filePath: "..\/outside.txt", content: "x" } })).rejects.toThrow(/native-path:outside-project/);
    } finally {
      if (previous === undefined) delete process.env.CODEA_HOME;
      else process.env.CODEA_HOME = previous;
      fs.rmSync(tmp, { recursive: true, force: true });
    }
  });

  test("corrupt persisted state fails closed until a valid replacement plan is created", async () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-task29-corrupt-"));
    const root = path.join(tmp, "project");
    const home = path.join(tmp, "home");
    fs.mkdirSync(root, { recursive: true });
    fs.mkdirSync(home, { recursive: true });
    const previous = process.env.CODEA_HOME;
    process.env.CODEA_HOME = home;
    try {
      let hooks = await plugin.server(input(root), { auditLog: path.join(tmp, "audit.log") });
      const ctx = context(root, "session-corrupt");
      await hooks.tool!.task_plan!.execute(plan, ctx);
      fs.writeFileSync(stateFile(home, root, "session-corrupt"), "{not-json\n", "utf8");

      hooks = await plugin.server(input(root), { auditLog: path.join(tmp, "audit-restart.log") });
      const corruptStatus = await hooks.tool!.task_status!.execute({}, ctx);
      expect(corruptStatus.metadata?.ok).toBe(false);
      expect(corruptStatus.metadata?.errorCategory).toBe("TASK_STATE_CORRUPT");
      expect(corruptStatus.metadata?.codeaTaskPlan).toBeUndefined();
      expect(corruptStatus.output).toContain("TASK_STATE_CORRUPT");
      await expect(hooks["tool.execute.before"]!({ tool: "write", sessionID: "session-corrupt", callID: "c1" }, { args: { filePath: "safe.txt", content: "x" } })).rejects.toThrow(/PLAN_REQUIRED/);

      const replaced = await hooks.tool!.task_plan!.execute(plan, ctx);
      expect(replaced.metadata?.codeaTaskPlan).toBe("true");
      await expect(hooks["tool.execute.before"]!({ tool: "write", sessionID: "session-corrupt", callID: "c2" }, { args: { filePath: "..\/outside.txt", content: "x" } })).rejects.toThrow(/native-path:outside-project/);
    } finally {
      if (previous === undefined) delete process.env.CODEA_HOME;
      else process.env.CODEA_HOME = previous;
      fs.rmSync(tmp, { recursive: true, force: true });
    }
  });
});