import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { plugin } from "../src/opencode/entry";
import type { PluginInput, ToolContext } from "../src/opencode/types";

function makeInput(root: string): PluginInput {
  return {
    client: {}, project: {}, directory: root, worktree: root, experimental_workspace: {},
    serverUrl: new URL("http://127.0.0.1:4096"), $: {},
  } as unknown as PluginInput;
}

function toolContext(root: string, sessionID: string, messageID: string): ToolContext {
  return {
    sessionID, messageID, agent: "general", directory: root, worktree: root,
    abort: new AbortController().signal, metadata() {}, async ask() {},
  };
}

function planInput() {
  return {
    goal: "Bind mutations to the current root turn",
    steps: [
      { id: "inspect", title: "Inspect" },
      { id: "change", title: "Change" },
      { id: "verify", title: "Verify" },
    ],
  };
}

async function chatMessage(
  hooks: Awaited<ReturnType<typeof plugin.server>>,
  sessionID: string,
  messageID: string,
  role: string,
  parts: any[] = [],
) {
  const chat = (hooks as any)["chat.message"];
  expect(typeof chat).toBe("function");
  await chat(
    { sessionID, messageID, agent: "general" },
    { message: { id: messageID, sessionID, role }, parts },
  );
}

async function userMessage(hooks: Awaited<ReturnType<typeof plugin.server>>, sessionID: string, messageID: string, parts: any[] = []) {
  await chatMessage(hooks, sessionID, messageID, "user", parts);
}

async function expectPlanRequired(promise: Promise<unknown>) {
  try {
    await promise;
    throw new Error("expected PLAN_REQUIRED");
  } catch (error: any) {
    expect(error?.category).toBe("PLAN_REQUIRED");
  }
}

describe("Task 29 root-turn epoch contract", () => {
  test("a new ordinary user turn invalidates the prior root plan", async () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-task29-epoch-"));
    const root = path.join(tmp, "project");
    fs.mkdirSync(path.join(root, "src"), { recursive: true });
    const previousHome = process.env.CODEA_HOME;
    process.env.CODEA_HOME = path.join(tmp, "home");
    try {
      const hooks = await plugin.server(makeInput(root), { auditLog: path.join(tmp, "audit.log") });
      await userMessage(hooks, "s1", "U1");
      const plan = await hooks.tool!.task_plan!.execute(planInput(), toolContext(root, "s1", "A1"));
      expect((plan as any).metadata?.ok).toBe(true);

      await expect(hooks["tool.execute.before"]!({ tool: "write", sessionID: "s1", callID: "w1" }, { args: { filePath: "src/Foo.java", content: "class Foo {}" } })).resolves.toBeUndefined();

      await userMessage(hooks, "s1", "U2");
      await expectPlanRequired(hooks["tool.execute.before"]!({ tool: "write", sessionID: "s1", callID: "w2" }, { args: { filePath: "src/Bar.java", content: "class Bar {}" } }));
      console.log("NEW_USER_TURN_INVALIDATES_PRIOR_PLAN PASS");
    } finally {
      if (previousHome === undefined) delete process.env.CODEA_HOME; else process.env.CODEA_HOME = previousHome;
      fs.rmSync(tmp, { recursive: true, force: true });
    }
  });

  test("assistant messages inside the same root turn do not invalidate its plan", async () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-task29-assistant-epoch-"));
    const root = path.join(tmp, "project");
    fs.mkdirSync(path.join(root, "src"), { recursive: true });
    const previousHome = process.env.CODEA_HOME;
    process.env.CODEA_HOME = path.join(tmp, "home");
    try {
      const hooks = await plugin.server(makeInput(root), { auditLog: path.join(tmp, "audit.log") });
      await userMessage(hooks, "s1", "U1");
      const plan = await hooks.tool!.task_plan!.execute(planInput(), toolContext(root, "s1", "A1"));
      expect((plan as any).metadata?.ok).toBe(true);

      // OpenCode emits chat.message for assistant/tool-loop messages after the
      // plan tool returns. Those are continuations of U1, not new root turns.
      await chatMessage(hooks, "s1", "A2", "assistant");

      await expect(hooks["tool.execute.before"]!({ tool: "write", sessionID: "s1", callID: "w-assistant" }, { args: { filePath: "src/Assistant.java", content: "class Assistant {}" } })).resolves.toBeUndefined();
      console.log("ASSISTANT_MESSAGE_PRESERVES_ROOT_EPOCH PASS");
    } finally {
      if (previousHome === undefined) delete process.env.CODEA_HOME; else process.env.CODEA_HOME = previousHome;
      fs.rmSync(tmp, { recursive: true, force: true });
    }
  });

  test("verification-control continuation preserves the original root epoch", async () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-task29-continuation-"));
    const root = path.join(tmp, "project");
    fs.mkdirSync(path.join(root, "src"), { recursive: true });
    const previousHome = process.env.CODEA_HOME;
    process.env.CODEA_HOME = path.join(tmp, "home");
    try {
      const hooks = await plugin.server(makeInput(root), { auditLog: path.join(tmp, "audit.log") });
      await userMessage(hooks, "s1", "U1");
      const plan = await hooks.tool!.task_plan!.execute(planInput(), toolContext(root, "s1", "A1"));
      expect((plan as any).metadata?.ok).toBe(true);

      await userMessage(hooks, "s1", "C1", [{
        type: "text", synthetic: true, text: "verification continuation",
        metadata: { "codea.kind": "verification-control", "codea.rootTurn": "U1" },
      }]);

      await expect(hooks["tool.execute.before"]!({ tool: "write", sessionID: "s1", callID: "w3" }, { args: { filePath: "src/Foo.java", content: "class Foo {}" } })).resolves.toBeUndefined();
      console.log("CONTROL_CONTINUATION_PRESERVES_ROOT_EPOCH PASS");
    } finally {
      if (previousHome === undefined) delete process.env.CODEA_HOME; else process.env.CODEA_HOME = previousHome;
      fs.rmSync(tmp, { recursive: true, force: true });
    }
  });
});
