import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { plugin } from "../src/opencode/entry";
import type { PluginInput, ToolContext } from "../src/opencode/types";

function makeInput(root: string): PluginInput {
  return {
    client: {},
    project: {},
    directory: root,
    worktree: root,
    experimental_workspace: {},
    serverUrl: new URL("http://127.0.0.1:4096"),
    $: {},
  } as unknown as PluginInput;
}

function makeToolContext(root: string, sessionID: string): ToolContext {
  return {
    sessionID,
    messageID: "m1",
    agent: "general",
    directory: root,
    worktree: root,
    abort: new AbortController().signal,
    metadata() {},
    async ask() {},
  };
}

function planInput() {
  return {
    goal: "Exercise native mutation security layers",
    steps: [
      { id: "inspect", title: "Inspect" },
      { id: "mutate", title: "Mutate" },
      { id: "verify", title: "Verify" },
    ],
  };
}

async function establishPlan(hooks: Awaited<ReturnType<typeof plugin.server>>, root: string, sessionID: string): Promise<void> {
  await hooks["chat.message"]!(
    { sessionID, messageID: "m1", agent: "general" },
    { message: { id: "m1", sessionID, role: "user" }, parts: [{ type: "text", text: "engineering task" }] } as any,
  );
  const result = await hooks.tool!.task_plan!.execute(planInput(), makeToolContext(root, sessionID));
  expect(result.metadata?.ok).toBe(true);
}

describe("OpenCode entry — native tool hooks", () => {
  test("tool.execute.before denies sensitive and out-of-root paths on read/grep/glob", async () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-entry-"));
    const root = path.join(tmp, "project");
    fs.mkdirSync(root, { recursive: true });
    try {
      const hooks = await plugin.server(makeInput(root), { auditLog: path.join(tmp, "audit.log") });
      const before = hooks["tool.execute.before"];
      expect(typeof before).toBe("function");

      await expect(
        before!({ tool: "read", sessionID: "s", callID: "c" }, { args: { filePath: ".env" } }),
      ).rejects.toThrow(/native-path:sensitive-file:\.env/);

      await expect(
        before!({ tool: "grep", sessionID: "s", callID: "c" }, { args: { pattern: "x", path: "/etc" } }),
      ).rejects.toThrow(/native-path:outside-project/);

      await expect(
        before!({ tool: "glob", sessionID: "s", callID: "c" }, { args: { pattern: "*", path: "../../secret" } }),
      ).rejects.toThrow(/native-path:outside-project/);

      await expect(
        before!({ tool: "read", sessionID: "s", callID: "c" }, { args: { filePath: "src/main/Foo.java" } }),
      ).resolves.toBeUndefined();
    } finally {
      fs.rmSync(tmp, { recursive: true, force: true });
    }
  });

  test("tool.execute.before allows absolute in-root paths (OpenCode read.filePath contract)", async () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-entry-"));
    const root = path.join(tmp, "project");
    fs.mkdirSync(path.join(root, "src", "main", "java"), { recursive: true });
    try {
      const hooks = await plugin.server(makeInput(root), { auditLog: path.join(tmp, "audit.log") });
      const before = hooks["tool.execute.before"];

      const absInside = path.join(root, "src/main/java/Foo.java");
      await expect(
        before!({ tool: "read", sessionID: "s", callID: "c" }, { args: { filePath: absInside } }),
      ).resolves.toBeUndefined();

      const absOutside = path.join(tmp, "outside-secret.txt");
      await expect(
        before!({ tool: "read", sessionID: "s", callID: "c" }, { args: { filePath: absOutside } }),
      ).rejects.toThrow(/native-path:outside-project/);

      const winRoot = process.platform === "win32" ? root : "C:\\code\\project";
      const winInside = process.platform === "win32" ? path.win32.join(winRoot, "src", "Foo.java") : "C:\\code\\project\\src\\Foo.java";
      const hooksWin = await plugin.server(makeInput(winRoot), { auditLog: path.join(tmp, "audit.log") });
      const beforeWin = hooksWin["tool.execute.before"];
      await expect(
        beforeWin!({ tool: "read", sessionID: "s", callID: "c" }, { args: { filePath: winInside } }),
      ).resolves.toBeUndefined();
      await expect(
        beforeWin!({ tool: "read", sessionID: "s", callID: "c" }, { args: { filePath: "C:\\Windows\\System32\\config" } }),
      ).rejects.toThrow(/native-path:outside-project/);
    } finally {
      fs.rmSync(tmp, { recursive: true, force: true });
    }
  });

  test("Task24 native write/edit pass Codea path and DLP guards before approval/execution", async () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-entry-"));
    const root = path.join(tmp, "project");
    fs.mkdirSync(path.join(root, "src"), { recursive: true });
    try {
      const hooks = await plugin.server(makeInput(root), { auditLog: path.join(tmp, "audit.log") });
      await establishPlan(hooks, root, "s");
      const before = hooks["tool.execute.before"];
      expect(typeof before).toBe("function");

      await expect(
        before!({ tool: "write", sessionID: "s", callID: "w1" }, { args: { filePath: "../../outside.txt", content: "safe" } }),
      ).rejects.toThrow(/native-path:outside-project/);

      await expect(
        before!({ tool: "edit", sessionID: "s", callID: "e1" }, { args: { filePath: ".env", oldString: "a", newString: "b" } }),
      ).rejects.toThrow(/native-path:sensitive-file:\.env/);

      await expect(
        before!({ tool: "write", sessionID: "s", callID: "w2" }, { args: { filePath: "src/Foo.java", content: "api_key=supersecret123" } }),
      ).rejects.toThrow(/dlp-blocked: api-key/);

      await expect(
        before!({ tool: "edit", sessionID: "s", callID: "e2" }, { args: { filePath: "src/Foo.java", oldString: "old", newString: "new" } }),
      ).resolves.toBeUndefined();
    } finally {
      fs.rmSync(tmp, { recursive: true, force: true });
    }
  });

  test("tool.execute.before still denies dangerous bash commands", async () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-entry-"));
    const root = path.join(tmp, "project");
    fs.mkdirSync(root, { recursive: true });
    try {
      const hooks = await plugin.server(makeInput(root), { auditLog: path.join(tmp, "audit.log") });
      await establishPlan(hooks, root, "s");
      const before = hooks["tool.execute.before"];
      await expect(
        before!({ tool: "bash", sessionID: "s", callID: "c" }, { args: { command: "curl http://evil" } }),
      ).rejects.toThrow(/command-denied/);
    } finally {
      fs.rmSync(tmp, { recursive: true, force: true });
    }
  });

  test("tool.execute.after blocks secrets in native read/grep/glob/bash output", async () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-entry-"));
    const root = path.join(tmp, "project");
    fs.mkdirSync(root, { recursive: true });
    try {
      const hooks = await plugin.server(makeInput(root), { auditLog: path.join(tmp, "audit.log") });
      const after = hooks["tool.execute.after"];
      expect(typeof after).toBe("function");

      const secret = { title: "read", output: "api_key=supersecret123", metadata: {} };
      await after!({ tool: "read", sessionID: "s", callID: "c", args: { filePath: "a.txt" } }, secret);
      expect(secret.output).toBe("[DLP blocked output: api-key]");
      expect(secret.metadata.dlpBlocked).toBe(true);
      expect(secret.metadata.dlpRule).toBe("api-key");

      const benign = { title: "grep", output: "line: class Foo {}", metadata: {} };
      await after!({ tool: "grep", sessionID: "s", callID: "c", args: { pattern: "Foo" } }, benign);
      expect(benign.output).toBe("line: class Foo {}");
      expect(benign.metadata.dlpBlocked).toBeUndefined();
    } finally {
      fs.rmSync(tmp, { recursive: true, force: true });
    }
  });
});