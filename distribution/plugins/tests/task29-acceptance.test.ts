import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { plugin } from "../src/opencode/entry";
import type { PluginInput, ToolContext } from "../src/opencode/types";

function input(root: string): PluginInput {
  return {
    client: {}, project: {}, directory: root, worktree: root,
    experimental_workspace: {}, serverUrl: new URL("http://127.0.0.1:4096"), $: {},
  } as unknown as PluginInput;
}

function rootID(sessionID: string): string {
  return `turn-${sessionID}`;
}

function context(root: string, sessionID: string): ToolContext {
  return {
    sessionID, messageID: rootID(sessionID), agent: "general", directory: root, worktree: root,
    abort: new AbortController().signal, metadata() {}, async ask() {},
  };
}

async function establishRoot(hooks: Awaited<ReturnType<typeof plugin.server>>, sessionID: string): Promise<void> {
  await hooks["chat.message"]!(
    { sessionID, messageID: rootID(sessionID), agent: "general" },
    { message: { id: rootID(sessionID), sessionID, role: "user" }, parts: [{ type: "text", text: "engineering task" }] } as any,
  );
}

const plan = {
  goal: "Prove Task 29 protocol",
  steps: [
    { id: "inspect", title: "Inspect evidence" },
    { id: "change", title: "Apply change" },
    { id: "verify", title: "Verify result" },
  ],
};

async function expectPlanRequired(promise: Promise<unknown>): Promise<void> {
  try {
    await promise;
    throw new Error("expected PLAN_REQUIRED");
  } catch (error: any) {
    expect(error?.code).toBe("PLAN_REQUIRED");
    expect(error?.category).toBe("PLAN_REQUIRED");
  }
}

describe("Task 29 mechanical acceptance", () => {
  test("protocol markers are backed by real plugin hook/tool behavior", async () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-task29-acceptance-"));
    const root = path.join(tmp, "project");
    const home = path.join(tmp, "home");
    fs.mkdirSync(path.join(root, "docs"), { recursive: true });
    fs.mkdirSync(home, { recursive: true });
    const previous = process.env.CODEA_HOME;
    process.env.CODEA_HOME = home;
    try {
      const sessionA = "session-A";
      const sessionB = "session-B";
      const first = await plugin.server(input(root), { auditLog: path.join(tmp, "audit-1.log") });
      await establishRoot(first, sessionA);
      await establishRoot(first, sessionB);
      const before = first["tool.execute.before"]!;

      await expect(before({ tool: "read", sessionID: sessionA, callID: "read-1" }, { args: { filePath: "docs/input.md" } })).resolves.toBeUndefined();
      console.log("READ_WITHOUT_PLAN PASS");

      await expectPlanRequired(before({ tool: "write", sessionID: sessionA, callID: "write-1" }, { args: { filePath: "docs/out.md", content: "x" } }));
      console.log("WRITE_WITHOUT_PLAN PLAN_REQUIRED");

      await expectPlanRequired(before({ tool: "edit", sessionID: sessionA, callID: "edit-1" }, { args: { filePath: "docs/out.md", oldString: "x", newString: "y" } }));
      console.log("EDIT_WITHOUT_PLAN PLAN_REQUIRED");

      await expectPlanRequired(before({ tool: "bash", sessionID: sessionA, callID: "bash-1" }, { args: { command: "echo safe" } }));
      console.log("BASH_WITHOUT_PLAN PLAN_REQUIRED");

      await expectPlanRequired(first.tool!.write_document!.execute({ path: "docs/out.md", content: "x" }, context(root, sessionA)));
      console.log("ENTERPRISE_WRITE_WITHOUT_PLAN PLAN_REQUIRED");

      const created = await first.tool!.task_plan!.execute(plan, context(root, sessionA));
      expect(created.metadata?.ok).toBe(true);
      expect(created.metadata?.codeaTaskPlan).toBe("true");
      expect(created.metadata?.codeaPlanTotal).toBe("3");
      console.log("PLAN_3_TO_7_STEPS PASS");

      const active = await first.tool!.task_step!.execute({ stepId: "inspect", status: "in_progress" }, context(root, sessionA));
      expect(active.metadata?.ok).toBe(true);
      expect(active.metadata?.codeaPlanActive).toBe("inspect");
      const secondActive = await first.tool!.task_step!.execute({ stepId: "change", status: "in_progress" }, context(root, sessionA));
      expect(secondActive.metadata?.ok).toBe(false);
      expect(secondActive.metadata?.errorCategory).toBe("TASK_STATE_INVALID");
      console.log("SINGLE_ACTIVE_STEP PASS");

      await expectPlanRequired(before({ tool: "write", sessionID: sessionB, callID: "write-b" }, { args: { filePath: "docs/b.md", content: "x" } }));
      console.log("CROSS_SESSION_PLAN_ISOLATION PASS");

      const restarted = await plugin.server(input(root), { auditLog: path.join(tmp, "audit-2.log") });
      await establishRoot(restarted, sessionA);
      const restored = await restarted.tool!.task_status!.execute({}, context(root, sessionA));
      expect(restored.metadata?.ok).toBe(true);
      expect(restored.metadata?.codeaTaskPlan).toBe("true");
      expect(restored.metadata?.codeaPlanTotal).toBe("3");
      expect(restored.metadata?.codeaPlanActive).toBe("inspect");
      console.log("PLAN_PERSISTENCE PASS");
    } finally {
      if (previous === undefined) delete process.env.CODEA_HOME;
      else process.env.CODEA_HOME = previous;
      fs.rmSync(tmp, { recursive: true, force: true });
    }
  });
});
