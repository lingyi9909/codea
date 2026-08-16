import { describe, expect, test } from "bun:test";
import * as path from "node:path";
import { analyzeTestProjectTool } from "../src/tools/analyze-test-project";
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
