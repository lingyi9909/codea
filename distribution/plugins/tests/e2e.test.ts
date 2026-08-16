import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as path from "node:path";
import { makeTempRoot, makeContext } from "./helpers";
import { execCommand } from "../src/tools/exec";
import { collectReviewContextTool } from "../src/tools/collect-review-context";
import { analyzeTestProjectTool } from "../src/tools/analyze-test-project";
import { writeTestFileTool } from "../src/tools/write-test-file";
import { runProjectTestTool } from "../src/tools/run-project-test";
import { extractApiSpecTool } from "../src/tools/extract-api-spec";
import { validateApiExampleTool } from "../src/tools/validate-api-example";
import { writeDocumentTool } from "../src/tools/write-document";

const FIXTURE = path.resolve(import.meta.dir, "../../../tui/tests/e2e/fixtures/java-maven-project");

function copyFixture(prefix: string): string {
  const root = makeTempRoot(prefix);
  fs.cpSync(FIXTURE, root, { recursive: true });
  return root;
}

describe("E2E — Review Tool Flow", () => {
  test("git change -> collect_review_context -> accurate diff/line/file", async () => {
    const root = makeTempRoot("codea-e2e-review-");
    const { ctx } = makeContext(root);

    await execCommand(["git", "init", "-q"], { cwd: root });
    await execCommand(["git", "config", "user.email", "t@example.com"], { cwd: root });
    await execCommand(["git", "config", "user.name", "t"], { cwd: root });

    fs.writeFileSync(path.join(root, "A.java"), "line1\nline2\n");
    await execCommand(["git", "add", "."], { cwd: root });
    await execCommand(["git", "commit", "-qm", "init"], { cwd: root });

    fs.writeFileSync(path.join(root, "A.java"), "line1\nline2\nline3\n");

    const result = await collectReviewContextTool.execute({ source: "unstaged" }, ctx);
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data.filesChanged).toBe(1);
      expect(result.data.linesAdded).toBe(1);
      expect(result.data.files[0].path).toBe("A.java");
      expect(result.data.files[0].hunks.length).toBe(1);
    }
  });
});

describe("E2E — Unit Test Tool Flow", () => {
  test("analyze -> write_test_file -> run_project_test -> structured result", async () => {
    const root = copyFixture("codea-e2e-unit-");
    const { ctx } = makeContext(root);

    const analyze = await analyzeTestProjectTool.execute({}, ctx);
    expect(analyze.ok).toBe(true);
    if (!analyze.ok) return;
    expect(analyze.data.buildSystem).toBe("maven");
    expect(analyze.data.testRoots).toContain("src/test/java");

    const write = await writeTestFileTool.execute(
      { path: "src/test/java/com/example/demo/NewFeatureTest.java", content: "package com.example.demo;\nclass NewFeatureTest {}\n" },
      ctx,
    );
    expect(write.ok).toBe(true);

    const run = await runProjectTestTool.execute({ buildSystem: "maven" }, ctx);
    expect(run.ok).toBe(true);
    if (run.ok) {
      expect(run.data.category).toBe("PASS");
      expect(run.data.passed).toBe(3);
    }
  });
});

describe("E2E — API Doc Tool Flow", () => {
  test("extract_api_spec -> validate_api_example -> write_document -> markdown on disk", async () => {
    const root = copyFixture("codea-e2e-api-");
    const { ctx } = makeContext(root);

    const extract = await extractApiSpecTool.execute(
      { controllerFile: "src/main/java/com/example/demo/DemoController.java" },
      ctx,
    );
    expect(extract.ok).toBe(true);
    if (!extract.ok) return;

    const postIndex = extract.data.endpoints.findIndex((e) => e.method === "POST");
    expect(postIndex).toBeGreaterThanOrEqual(0);

    const validate = await validateApiExampleTool.execute(
      { example: { name: "Alice", email: "a@example.com", age: 30 }, spec: extract.data, endpointIndex: postIndex },
      ctx,
    );
    expect(validate.ok).toBe(true);
    if (validate.ok) expect(validate.data.valid).toBe(true);

    const markdown = `# User API\n\nBase path: ${extract.data.basePath}\n\n## POST\n\nCreates a user.\n`;
    const write = await writeDocumentTool.execute({ path: "docs/api/users.md", content: markdown }, ctx);
    expect(write.ok).toBe(true);
    expect(fs.existsSync(path.join(root, "docs/api/users.md"))).toBe(true);
    expect(fs.readFileSync(path.join(root, "docs/api/users.md"), "utf8")).toContain("/api/users");
  });
});
