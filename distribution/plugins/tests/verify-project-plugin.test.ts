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

function context(root: string, asks: { count: number }): ToolContext {
  return {
    sessionID: "s1", messageID: "u1", agent: "general", directory: root, worktree: root,
    abort: new AbortController().signal,
    metadata() {},
    async ask() { asks.count += 1; },
  };
}

async function establishRootAndPlan(hooks: Awaited<ReturnType<typeof plugin.server>>, root: string): Promise<void> {
  await hooks["chat.message"]!(
    { sessionID: "s1", messageID: "u1", agent: "general" },
    { message: { id: "u1", sessionID: "s1", role: "user" }, parts: [{ type: "text", text: "mutating task" }] } as any,
  );
  const planned = await hooks.tool!.task_plan!.execute({
    goal: "Mutate then verify",
    steps: [
      { id: "inspect", title: "Inspect" },
      { id: "change", title: "Change" },
      { id: "verify", title: "Verify" },
    ],
  }, context(root, { count: 0 }));
  expect(planned.metadata?.ok).toBe(true);
}

async function expectPlanRequired(promise: Promise<unknown>): Promise<void> {
  try {
    await promise;
    throw new Error("expected PLAN_REQUIRED");
  } catch (error: any) {
    expect(error?.code).toBe("PLAN_REQUIRED");
    expect(error?.category).toBe("PLAN_REQUIRED");
  }
}

describe("verify_project plugin registration and security", () => {
  test("registers bounded schema with no arbitrary command, args, root, or repository", async () => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "codea-verify-plugin-"));
    try {
      const hooks = await plugin.server(input(root), { auditLog: path.join(root, "audit.log") });
      const verify = hooks.tool?.verify_project;
      expect(verify).toBeDefined();
      expect(Object.keys(verify!.args)).toEqual(["timeoutSeconds"]);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });

  test("PLAN_REQUIRED is enforced before verification execution", async () => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "codea-verify-plugin-"));
    const asks = { count: 0 };
    try {
      const hooks = await plugin.server(input(root), { auditLog: path.join(root, "audit.log") });
      await hooks["chat.message"]!(
        { sessionID: "s1", messageID: "u1", agent: "general" },
        { message: { id: "u1", sessionID: "s1", role: "user" }, parts: [{ type: "text", text: "mutating task" }] } as any,
      );
      await expectPlanRequired(hooks.tool!.verify_project!.execute({}, context(root, asks)));
      expect(asks.count).toBe(0);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });

  test("plan-authorized verification still uses approval and emits only allowlisted verification metadata", async () => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "codea-verify-plugin-"));
    const asks = { count: 0 };
    try {
      await establishRootAndPlan(await plugin.server(input(root), { auditLog: path.join(root, "unused.log") }), root);
      const hooks = await plugin.server(input(root), { auditLog: path.join(root, "audit.log") });
      await establishRootAndPlan(hooks, root);
      const out = await hooks.tool!.verify_project!.execute({}, context(root, asks)) as any;
      expect(asks.count).toBe(1);
      expect(out.metadata.codeaVerification).toBe("true");
      expect(out.metadata.codeaVerificationResult).toBe("not_configured");
      expect(out.metadata.codeaVerificationProfile).toBe("unknown");
      expect(out.metadata.stages).toBeUndefined();
      expect(out.metadata.evidence).toBeUndefined();
      expect(out.metadata.command).toBeUndefined();
      expect(out.output).toContain("NOT_CONFIGURED");
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });

  test("verification command output remains DLP-safe through returned evidence", async () => {
    if (process.platform === "win32") return;
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "codea-verify-plugin-"));
    const asks = { count: 0 };
    try {
      fs.writeFileSync(path.join(root, "pom.xml"), "<project><modelVersion>4.0.0</modelVersion></project>", "utf8");
      const wrapper = path.join(root, "mvnw");
      fs.writeFileSync(wrapper, "#!/bin/sh\necho 'api_key=supersecret123'\nexit 1\n", { encoding: "utf8", mode: 0o755 });
      const hooks = await plugin.server(input(root), { auditLog: path.join(root, "audit.log") });
      await establishRootAndPlan(hooks, root);
      const out = await hooks.tool!.verify_project!.execute({}, context(root, asks)) as any;
      expect(out.output).not.toContain("supersecret123");
      expect(out.output).toContain("[REDACTED]");
      expect(out.metadata.codeaVerificationResult).toBe("fail");
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });
});
