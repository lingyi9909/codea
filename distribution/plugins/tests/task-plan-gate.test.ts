import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { plugin } from "../src/opencode/entry";
import type { PluginInput, ToolContext as OpenCodeToolContext } from "../src/opencode/types";

function makeInput(root: string): PluginInput {
  return {
    client: {}, project: {}, directory: root, worktree: root, experimental_workspace: {},
    serverUrl: new URL("http://127.0.0.1:4096"), $: {},
  } as unknown as PluginInput;
}

function makeContext(root: string, sessionID: string, askEvents: any[] = []): OpenCodeToolContext {
  return {
    sessionID,
    messageID: "m1",
    agent: "general",
    directory: root,
    worktree: root,
    abort: new AbortController().signal,
    metadata() {},
    async ask(input) { askEvents.push(input); },
  };
}

function planInput() {
  return {
    goal: "Mutate project safely",
    steps: [
      { id: "inspect", title: "Inspect evidence" },
      { id: "change", title: "Apply change" },
      { id: "verify", title: "Verify result" },
    ],
  };
}

async function withPlugin(run: (ctx: { root: string; hooks: Awaited<ReturnType<typeof plugin.server>>; octx: OpenCodeToolContext }) => Promise<void>) {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-task29-gate-"));
  const root = path.join(tmp, "project");
  const home = path.join(tmp, "home");
  fs.mkdirSync(path.join(root, "src"), { recursive: true });
  fs.mkdirSync(path.join(root, "docs"), { recursive: true });
  fs.mkdirSync(home, { recursive: true });
  const previous = process.env.CODEA_HOME;
  process.env.CODEA_HOME = home;
  try {
    const hooks = await plugin.server(makeInput(root), { auditLog: path.join(tmp, "audit.log") });
    await run({ root, hooks, octx: makeContext(root, "session-gate") });
  } finally {
    if (previous === undefined) delete process.env.CODEA_HOME;
    else process.env.CODEA_HOME = previous;
    fs.rmSync(tmp, { recursive: true, force: true });
  }
}

async function expectPlanRequired(promise: Promise<unknown>) {
  try {
    await promise;
    throw new Error("expected PLAN_REQUIRED");
  } catch (error: any) {
    expect(error?.code).toBe("PLAN_REQUIRED");
    expect(error?.category).toBe("PLAN_REQUIRED");
  }
}

describe("Task 29 plan-before-mutation gate", () => {
  test("read/grep/glob remain available without a plan", async () => {
    await withPlugin(async ({ hooks }) => {
      const before = hooks["tool.execute.before"]!;
      await expect(before({ tool: "read", sessionID: "s", callID: "r" }, { args: { filePath: "src/Foo.java" } })).resolves.toBeUndefined();
      await expect(before({ tool: "grep", sessionID: "s", callID: "g" }, { args: { pattern: "Foo", path: "src" } })).resolves.toBeUndefined();
      await expect(before({ tool: "glob", sessionID: "s", callID: "l" }, { args: { pattern: "*.java", path: "src" } })).resolves.toBeUndefined();
    });
  });

  test("native write/edit/bash fail with machine PLAN_REQUIRED before existing policies", async () => {
    await withPlugin(async ({ hooks }) => {
      const before = hooks["tool.execute.before"]!;
      await expectPlanRequired(before({ tool: "write", sessionID: "s", callID: "w" }, { args: { filePath: "../../outside.txt", content: "safe" } }));
      await expectPlanRequired(before({ tool: "edit", sessionID: "s", callID: "e" }, { args: { filePath: ".env", oldString: "a", newString: "b" } }));
      await expectPlanRequired(before({ tool: "bash", sessionID: "s", callID: "b" }, { args: { command: "curl http://evil" } }));
    });
  });

  test("task_plan stays available and a valid plan lets native tools reach existing path/command/DLP checks", async () => {
    await withPlugin(async ({ hooks, root }) => {
      const sessionID = "session-native";
      const octx = makeContext(root, sessionID);
      const planResult = await hooks.tool!.task_plan!.execute(planInput(), octx);
      expect(planResult.metadata?.ok).toBe(true);

      const before = hooks["tool.execute.before"]!;
      await expect(
        before({ tool: "write", sessionID, callID: "w" }, { args: { filePath: "../../outside.txt", content: "safe" } }),
      ).rejects.toThrow(/native-path:outside-project/);
      await expect(
        before({ tool: "edit", sessionID, callID: "e" }, { args: { filePath: "src/Foo.java", oldString: "a", newString: "b" } }),
      ).resolves.toBeUndefined();
      await expect(
        before({ tool: "bash", sessionID, callID: "b" }, { args: { command: "curl http://evil" } }),
      ).rejects.toThrow(/command-denied/);
      await expect(
        before({ tool: "write", sessionID, callID: "d" }, { args: { filePath: "src/Foo.java", content: "api_key=supersecret123" } }),
      ).rejects.toThrow(/dlp-blocked: api-key/);
    });
  });

  test("enterprise write/execute require plan before approval while read-only tools remain available", async () => {
    await withPlugin(async ({ hooks, root }) => {
      const askEvents: any[] = [];
      const octx = makeContext(root, "session-enterprise", askEvents);
      const read = await hooks.tool!.analyze_test_project!.execute({}, octx);
      expect(read.metadata?.errorCategory).not.toBe("PLAN_REQUIRED");

      await expectPlanRequired(hooks.tool!.write_document!.execute({ path: "docs/plan.md", content: "safe" }, octx));
      await expectPlanRequired(hooks.tool!.write_test_file!.execute({ path: "src/test/java/FooTest.java", content: "class FooTest {}" }, octx));
      await expectPlanRequired(hooks.tool!.run_project_test!.execute({ buildSystem: "maven" }, octx));
      expect(askEvents).toHaveLength(0);
    });
  });

  test("after plan enterprise write reaches existing approval and execution", async () => {
    await withPlugin(async ({ hooks, root }) => {
      const askEvents: any[] = [];
      const octx = makeContext(root, "session-enterprise-ok", askEvents);
      await hooks.tool!.task_plan!.execute(planInput(), octx);
      const result = await hooks.tool!.write_document!.execute({ path: "docs/plan.md", content: "safe" }, octx);
      expect(result.metadata?.ok).toBe(true);
      expect(askEvents).toHaveLength(1);
      expect(fs.readFileSync(path.join(root, "docs", "plan.md"), "utf8")).toBe("safe");
    });
  });
});
