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

describe("Task25 runtime trace evidence", () => {
  test("enterprise tool lifecycle emits explicit plugin execution evidence", async () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-task25-trace-"));
    const root = path.join(tmp, "project");
    fs.mkdirSync(root, { recursive: true });
    try {
      const hooks = await plugin.server(makeInput(root), { auditLog: path.join(tmp, "audit.log") });
      const after = hooks["tool.execute.after"];
      expect(typeof after).toBe("function");

      const output = { title: "collect_review_context", output: "ok", metadata: {} as Record<string, unknown> };
      await after!(
        { tool: "collect_review_context", sessionID: "s1", callID: "call-plugin-1", args: { source: "staged" } },
        output,
      );

      expect(output.metadata.codeaPlugin).toBe("codea-enterprise");
      expect(output.metadata.codeaPluginInvocationID).toBe("call-plugin-1");
    } finally {
      fs.rmSync(tmp, { recursive: true, force: true });
    }
  });
});
