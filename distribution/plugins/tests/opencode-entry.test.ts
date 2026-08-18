import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { plugin } from "../src/opencode/entry";
import type { PluginInput } from "../src/opencode/types";

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

describe("OpenCode entry — native tool hooks", () => {
  test("tool.execute.before denies sensitive paths on read/grep/glob", async () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-entry-"));
    const root = path.join(tmp, "project");
    fs.mkdirSync(root, { recursive: true });
    try {
      const hooks = await plugin.server(makeInput(root), { auditLog: path.join(tmp, "audit.log") });
      const before = hooks["tool.execute.before"];
      expect(typeof before).toBe("function");

      await expect(
        before!({ tool: "read", sessionID: "s", callID: "c" }, { args: { filePath: ".env" } }),
      ).rejects.toThrow(/sensitive-path/);

      await expect(
        before!({ tool: "grep", sessionID: "s", callID: "c" }, { args: { pattern: "x", path: "/etc" } }),
      ).rejects.toThrow(/sensitive-path/);

      await expect(
        before!({ tool: "glob", sessionID: "s", callID: "c" }, { args: { pattern: "*", path: "../../secret" } }),
      ).rejects.toThrow(/sensitive-path/);

      // benign path passes.
      await expect(
        before!({ tool: "read", sessionID: "s", callID: "c" }, { args: { filePath: "src/main/Foo.java" } }),
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
