import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { plugin } from "../src/opencode/entry";
import type { PluginInput, ToolContext } from "../src/opencode/types";

function input(root: string): PluginInput {
  return { client: {}, project: {}, directory: root, worktree: root, experimental_workspace: {}, serverUrl: new URL("http://127.0.0.1:4096"), $: {} } as unknown as PluginInput;
}

function context(root: string): ToolContext {
  return {
    sessionID: "session-metadata", messageID: "turn-1", agent: "general", directory: root, worktree: root,
    abort: new AbortController().signal, metadata() {}, async ask() {},
  };
}

const plan = {
  goal: "Implement safely",
  steps: [
    { id: "inspect", title: "Inspect evidence" },
    { id: "change", title: "Apply change" },
    { id: "verify", title: "Verify result" },
  ],
};

describe("Task 29 safe planning metadata", () => {
  test("task_plan and task_step emit only bounded execution state fields needed by Application", async () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-task29-metadata-"));
    const root = path.join(tmp, "project");
    const home = path.join(tmp, "home");
    fs.mkdirSync(root, { recursive: true });
    fs.mkdirSync(home, { recursive: true });
    const previous = process.env.CODEA_HOME;
    process.env.CODEA_HOME = home;
    try {
      const hooks = await plugin.server(input(root), { auditLog: path.join(tmp, "audit.log") });
      const octx = context(root);

      const created = await hooks.tool!.task_plan!.execute(plan, octx);
      expect(created.metadata?.codeaTaskPlan).toBe("true");
      expect(created.metadata?.codeaPlanTotal).toBe("3");
      expect(created.metadata?.codeaPlanCompleted).toBe("0");
      expect(created.metadata?.codeaPlanActive).toBe("");

      const active = await hooks.tool!.task_step!.execute({ stepId: "inspect", status: "in_progress" }, octx);
      expect(active.metadata?.codeaTaskPlan).toBe("true");
      expect(active.metadata?.codeaPlanTotal).toBe("3");
      expect(active.metadata?.codeaPlanCompleted).toBe("0");
      expect(active.metadata?.codeaPlanActive).toBe("inspect");

      const completed = await hooks.tool!.task_step!.execute({ stepId: "inspect", status: "completed", evidence: "inspection complete" }, octx);
      expect(completed.metadata?.codeaPlanCompleted).toBe("1");
      expect(completed.metadata?.codeaPlanActive).toBe("");
    } finally {
      if (previous === undefined) delete process.env.CODEA_HOME;
      else process.env.CODEA_HOME = previous;
      fs.rmSync(tmp, { recursive: true, force: true });
    }
  });
});
