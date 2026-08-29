import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import type { ExecResult } from "../src/tools/exec";
import type { ToolContext } from "../src/tools/types";

const verificationModule = await import("../src/tools/verify-project").catch(() => ({} as Record<string, unknown>));

type Runner = (argv: readonly string[], opts: { cwd: string; timeoutMs?: number }) => Promise<ExecResult>;
type VerifyTool = { execute(params: unknown, ctx: ToolContext): Promise<any> };
type CreateTool = (runner?: Runner) => VerifyTool;

function createTool(runner?: Runner): VerifyTool {
  expect(typeof verificationModule.createVerifyProjectTool).toBe("function");
  return (verificationModule.createVerifyProjectTool as CreateTool)(runner);
}

function goRoot(): string {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "codea-verify-exec-"));
  fs.writeFileSync(path.join(root, "go.mod"), "module example.com/verify\n\ngo 1.26\n", "utf8");
  return root;
}

function context(root: string): ToolContext {
  return {
    sessionId: "session-1",
    rootTurnId: "root-1",
    agent: "general",
    projectRoot: root,
    audit: {} as any,
    guard: { after() {} } as any,
  };
}

function result(argv: readonly string[], options: Partial<ExecResult> = {}): ExecResult {
  return {
    exitCode: 0,
    stdout: "ok",
    stderr: "",
    timedOut: false,
    command: argv.join(" "),
    ...options,
  };
}

describe("verify_project secure execution", () => {
  test("runs fixed detected stages and returns PASS evidence", async () => {
    const root = goRoot();
    const calls: string[][] = [];
    try {
      const tool = createTool(async (argv) => {
        calls.push([...argv]);
        return result(argv);
      });
      const out = await tool.execute({ timeoutSeconds: 5 }, context(root));
      expect(out.ok).toBe(true);
      expect(out.data.result).toBe("PASS");
      expect(out.data.profile).toBe("go");
      expect(out.data.stages.map((stage: any) => stage.name)).toEqual(["test", "vet"]);
      expect(calls).toEqual([["go", "test", "./..."], ["go", "vet", "./..."]]);
      expect(out.data.skippedStages).toEqual([]);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });

  test("nonzero stage fails and deterministically stops dependent stages", async () => {
    const root = goRoot();
    const calls: string[][] = [];
    try {
      const tool = createTool(async (argv) => {
        calls.push([...argv]);
        return result(argv, { exitCode: 1, stdout: "FAIL first stage" });
      });
      const out = await tool.execute({}, context(root));
      expect(out.ok).toBe(true);
      expect(out.data.result).toBe("FAIL");
      expect(out.data.stages).toHaveLength(1);
      expect(out.data.stages[0].category).toBe("FAIL");
      expect(out.data.skippedStages).toEqual([{ name: "vet", reason: "PRIOR_STAGE_NOT_PASS" }]);
      expect(calls).toEqual([["go", "test", "./..."]]);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });

  test("timeout maps to TIMEOUT and stops later stages", async () => {
    const root = goRoot();
    try {
      const tool = createTool(async (argv) => result(argv, { exitCode: 1, timedOut: true }));
      const out = await tool.execute({ timeoutSeconds: 1 }, context(root));
      expect(out.ok).toBe(true);
      expect(out.data.result).toBe("TIMEOUT");
      expect(out.data.stages[0].category).toBe("TIMEOUT");
      expect(out.data.skippedStages).toEqual([{ name: "vet", reason: "PRIOR_STAGE_NOT_PASS" }]);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });

  test("runner exception maps to ERROR evidence rather than throwing", async () => {
    const root = goRoot();
    try {
      const tool = createTool(async () => { throw new Error("spawn failed"); });
      const out = await tool.execute({}, context(root));
      expect(out.ok).toBe(true);
      expect(out.data.result).toBe("ERROR");
      expect(out.data.stages[0].category).toBe("ERROR");
      expect(out.data.stages[0].outputSummary).toContain("spawn failed");
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });

  test("unknown build is NOT_CONFIGURED and executes no command", async () => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "codea-verify-unknown-"));
    let calls = 0;
    try {
      const tool = createTool(async (argv) => {
        calls += 1;
        return result(argv);
      });
      const out = await tool.execute({}, context(root));
      expect(out.ok).toBe(true);
      expect(out.data.result).toBe("NOT_CONFIGURED");
      expect(out.data.profile).toBe("unknown");
      expect(out.data.stages).toEqual([]);
      expect(calls).toBe(0);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });

  test("stage summaries are bounded and redact secrets before returning evidence", async () => {
    const root = goRoot();
    try {
      const huge = `api_key=supersecret123 ${"x".repeat(8000)}`;
      const tool = createTool(async (argv) => result(argv, { exitCode: 1, stdout: huge }));
      const out = await tool.execute({}, context(root));
      const summary = out.data.stages[0].outputSummary as string;
      expect(summary).not.toContain("supersecret123");
      expect(summary).toContain("[REDACTED]");
      expect(summary.length).toBeLessThanOrEqual(2048);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });
});
