import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as path from "node:path";
import { analyzeTestProjectTool, detectTestRoots } from "../src/tools/analyze-test-project";
import { makeTempRoot, makeContext } from "./helpers";

const FIXTURE = path.resolve(import.meta.dir, "../../../tui/tests/e2e/fixtures/java-maven-project");

describe("analyzeTestProjectTool.execute", () => {
  test("detects maven + JUnit 5 + wrapper on the real fixture", async () => {
    const ctx = makeContext(FIXTURE).ctx;
    const result = await analyzeTestProjectTool.execute({}, ctx);
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data.buildSystem).toBe("maven");
      expect(result.data.testFramework).toBe("JUnit 5");
      expect(result.data.testRoots).toContain("src/test/java");
      expect(result.data.sourceRoots).toContain("src/main/java");
      expect(result.data.wrapperAvailable).toBe(true);
      expect(result.data.dependencies).toContain("mockito");
    }
  });

  test("returns NOT_SUPPORTED for an empty dir", async () => {
    const root = makeTempRoot("codea-analyze-");
    const ctx = makeContext(root).ctx;
    const result = await analyzeTestProjectTool.execute({}, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("NOT_SUPPORTED");
  });
});

describe("detectTestRoots — standard layout derivation", () => {
  test("derives src/test/java for a Maven project with no test dir yet", () => {
    const root = makeTempRoot("codea-roots-");
    fs.writeFileSync(path.join(root, "pom.xml"), "<project/>");
    expect(detectTestRoots(root)).toContain("src/test/java");
  });

  test("derives src/test/java and src/test/kotlin for a Gradle project", () => {
    const root = makeTempRoot("codea-roots-");
    fs.writeFileSync(path.join(root, "build.gradle"), "");
    const roots = detectTestRoots(root);
    expect(roots).toContain("src/test/java");
    expect(roots).toContain("src/test/kotlin");
  });

  test("returns empty for an unknown build system", () => {
    const root = makeTempRoot("codea-roots-");
    expect(detectTestRoots(root)).toEqual([]);
  });
});
