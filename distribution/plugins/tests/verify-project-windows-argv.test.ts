import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import type { ExecResult } from "../src/tools/exec";
import { resolveExecArgv } from "../src/tools/exec";
import type { ToolContext } from "../src/tools/types";
import { createVerifyProjectTool, detectVerificationProfile } from "../src/tools/verify-project";

function context(root: string): ToolContext {
  return {
    sessionId: "task30-windows-argv",
    rootTurnId: "root-windows-argv",
    agent: "debug",
    projectRoot: root,
    audit: {} as any,
    guard: { after() {} } as any,
  };
}

function pass(argv: readonly string[]): ExecResult {
  return {
    exitCode: 0,
    stdout: "ok",
    stderr: "",
    timedOut: false,
    command: argv.join(" "),
  };
}

describe("Task 30 Windows wrapper argv safety", () => {
  test("keeps a project path with spaces in cwd and routes fixed .cmd argv without shell interpolation", async () => {
    const parent = fs.mkdtempSync(path.join(os.tmpdir(), "codea-task30-windows-"));
    const root = path.join(parent, "project with spaces");
    fs.mkdirSync(root, { recursive: true });
    try {
      fs.writeFileSync(path.join(root, "pom.xml"), "<project><modelVersion>4.0.0</modelVersion></project>\n", "utf8");
      fs.writeFileSync(path.join(root, "mvnw.cmd"), "@echo off\r\nexit /b 0\r\n", "utf8");

      const profile = detectVerificationProfile(root, "win32");
      expect(profile.kind).toBe("maven");
      expect(profile.executable).toBe("./mvnw.cmd");
      expect(profile.stages[0]?.argv).toEqual(["./mvnw.cmd", "-DskipTests", "compile"]);
      expect(resolveExecArgv(profile.stages[0]!.argv, "win32")).toEqual([
        "cmd.exe",
        "/d",
        "/s",
        "/c",
        ".\\mvnw.cmd -DskipTests compile",
      ]);

      if (process.platform === "win32") {
        const calls: Array<{ argv: string[]; cwd: string }> = [];
        const tool = createVerifyProjectTool(async (argv, opts) => {
          calls.push({ argv: [...argv], cwd: opts.cwd });
          return pass(argv);
        });
        const out = await tool.execute({ timeoutSeconds: 5 }, context(root));
        expect(out.ok).toBe(true);
        expect(out.data.result).toBe("PASS");
        expect(calls).toEqual([
          { argv: ["./mvnw.cmd", "-DskipTests", "compile"], cwd: root },
          { argv: ["./mvnw.cmd", "test"], cwd: root },
        ]);
        for (const call of calls) {
          expect(call.argv.join(" ")).not.toContain(root);
          expect(call.argv.join(" ")).not.toMatch(/[&|<>^]/);
        }
      }
    } finally {
      fs.rmSync(parent, { recursive: true, force: true });
    }
  });

  test("routes fixed Gradle .bat argv through the same controlled Windows boundary", () => {
    expect(resolveExecArgv(["./gradlew.bat", "test"], "win32")).toEqual([
      "cmd.exe",
      "/d",
      "/s",
      "/c",
      ".\\gradlew.bat test",
    ]);
  });
});
