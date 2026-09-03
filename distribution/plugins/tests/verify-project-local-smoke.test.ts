import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import type { ToolContext } from "../src/tools/types";
import { createVerifyProjectTool } from "../src/tools/verify-project";

function context(root: string): ToolContext {
  return {
    sessionId: "task30-local-smoke",
    rootTurnId: "root-local-smoke",
    agent: "debug",
    projectRoot: root,
    audit: {} as any,
    guard: { after() {} } as any,
  };
}

function javaWrapperFixture(profile: "maven" | "gradle"): { base: string; root: string; log: string; wrapper: string } {
  const base = fs.mkdtempSync(path.join(os.tmpdir(), "codea-task30-java-smoke-"));
  const root = path.join(base, "project with spaces");
  fs.mkdirSync(root, { recursive: true });
  const log = path.join(root, ".wrapper-invocations.log");
  const windows = process.platform === "win32";
  const wrapper = profile === "maven" ? (windows ? "mvnw.cmd" : "mvnw") : windows ? "gradlew.bat" : "gradlew";
  fs.writeFileSync(path.join(root, profile === "maven" ? "pom.xml" : "build.gradle"), profile === "maven" ? "<project></project>\n" : "plugins { id 'java' }\n", "utf8");
  const stub = windows
    ? "@echo off\r\necho %*>>.wrapper-invocations.log\r\nexit /b 0\r\n"
    : "#!/bin/sh\nprintf '%s\\n' \"$*\" >> .wrapper-invocations.log\nexit 0\n";
  const wrapperPath = path.join(root, wrapper);
  fs.writeFileSync(wrapperPath, stub, "utf8");
  if (!windows) fs.chmodSync(wrapperPath, 0o755);
  return { base, root, log, wrapper };
}

describe("Task 30 real local verification smoke", () => {
  test("executes a standard-library-only Go project with the real runner", async () => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "codea-task30-go-smoke-"));
    try {
      fs.writeFileSync(path.join(root, "go.mod"), "module example.com/task30smoke\n\ngo 1.22\n", "utf8");
      fs.writeFileSync(path.join(root, "math.go"), "package smoke\n\nfunc Add(a, b int) int { return a + b }\n", "utf8");
      fs.writeFileSync(path.join(root, "math_test.go"), "package smoke\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) { if Add(2, 3) != 5 { t.Fatal(\"bad sum\") } }\n", "utf8");

      const out = await createVerifyProjectTool().execute({ timeoutSeconds: 30 }, context(root));
      expect(out.ok).toBe(true);
      expect(out.data.result).toBe("PASS");
      expect(out.data.profile).toBe("go");
      expect(out.data.stages.map((stage: any) => [stage.name, stage.category])).toEqual([
        ["test", "PASS"],
        ["vet", "PASS"],
      ]);
      expect(out.data.stages.map((stage: any) => stage.commandSummary)).toEqual([
        "go test ./...",
        "go vet ./...",
      ]);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  }, 60_000);

  test("executes Maven and Gradle stub wrappers through the real OS runner", async () => {
    for (const profile of ["maven", "gradle"] as const) {
      const fixture = javaWrapperFixture(profile);
      try {
        const out = await createVerifyProjectTool().execute({ timeoutSeconds: 30 }, context(fixture.root));
        expect(out.ok).toBe(true);
        expect(out.data.profile).toBe(profile);
        expect(out.data.result).toBe("PASS");
        const expectedStages = profile === "maven" ? ["compile", "test"] : ["classes", "test"];
        expect(out.data.stages.map((stage: any) => stage.name)).toEqual(expectedStages);
        expect(out.data.stages.map((stage: any) => stage.category)).toEqual(["PASS", "PASS"]);
        expect(out.data.skipped).toEqual([]);
        expect(out.data.stages.every((stage: any) => stage.commandSummary.startsWith(`./${fixture.wrapper}`))).toBe(true);

        const invocations = fs.readFileSync(fixture.log, "utf8").replace(/\r/g, "").trim().split("\n");
        expect(invocations).toEqual(profile === "maven" ? ["-DskipTests compile", "test"] : ["classes", "test"]);
      } finally {
        fs.rmSync(fixture.base, { recursive: true, force: true });
      }
    }
    console.log("REAL_JAVA_WRAPPER_SMOKE PASS");
  }, 60_000);
});
