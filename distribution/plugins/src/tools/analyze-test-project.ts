import { fileExists, listDir, readTextFile } from "./filesystem";
import { notSupported } from "./errors";
import { toToolError } from "./failure-classifier";
import { ok, err, type ToolContext, type ToolResult } from "./types";

// Unit Test tool: determines build system, test framework and test/source roots
// from the real project layout. Read-only. Never guesses a version — if a
// framework cannot be determined it reports "unknown" rather than inventing one.

export type BuildSystem = "maven" | "gradle" | "unknown";

export interface TestProjectInfo {
  buildSystem: BuildSystem;
  testFramework: string;
  testRoots: string[];
  sourceRoots: string[];
  wrapperAvailable: boolean;
  dependencies: string[];
  existingTestPattern: string;
}

const DEFAULT_SOURCE_ROOTS = ["src/main/java"];

function detectBuildSystem(root: string): BuildSystem {
  if (fileExists(root, "pom.xml")) return "maven";
  if (fileExists(root, "build.gradle") || fileExists(root, "build.gradle.kts")) return "gradle";
  return "unknown";
}

function detectTestFramework(root: string, buildSystem: BuildSystem): { framework: string; dependencies: string[] } {
  const candidates = buildSystem === "maven" ? ["pom.xml"] : ["build.gradle", "build.gradle.kts"];
  const dependencies: string[] = [];
  let text = "";
  for (const f of candidates) {
    if (fileExists(root, f)) {
      text += readTextFile(root, f);
    }
  }

  const framework = /junit-jupiter|junit\.jupiter|org\.junit\.jupiter/.test(text)
    ? "JUnit 5"
    : /junit/.test(text)
      ? "JUnit 4"
      : "unknown";

  if (/mockito/.test(text)) dependencies.push("mockito");
  if (/assertj/.test(text)) dependencies.push("assertj");
  if (/hamcrest/.test(text)) dependencies.push("hamcrest");
  if (/surefire/.test(text)) dependencies.push("maven-surefire-plugin");

  return { framework, dependencies };
}

function detectTestRoots(root: string): string[] {
  const roots: string[] = [];
  if (fileExists(root, "src/test/java")) roots.push("src/test/java");
  if (fileExists(root, "src/test/kotlin")) roots.push("src/test/kotlin");
  if (fileExists(root, "src/test/groovy")) roots.push("src/test/groovy");
  return roots;
}

function detectWrapper(root: string): boolean {
  return fileExists(root, "mvnw") || fileExists(root, "gradlew");
}

function detectExistingTestPattern(root: string, testRoots: string[]): string {
  for (const tr of testRoots) {
    let entries: string[] = [];
    try {
      entries = listDir(root + "/" + tr);
    } catch {
      continue;
    }
    if (entries.some((e) => e.endsWith("Test.java"))) return "Test.java";
    if (entries.some((e) => e.endsWith("Tests.java"))) return "Tests.java";
    if (entries.some((e) => e.endsWith("Test.kt"))) return "Test.kt";
  }
  return "Test.java";
}

export const analyzeTestProjectTool = {
  name: "analyze_test_project",
  description: "Analyze project structure to determine build system, test directories, and framework.",
  parameters: { type: "object", properties: {}, required: [] },

  async execute(_params: unknown, ctx: ToolContext): Promise<ToolResult<TestProjectInfo>> {
    const started = Date.now();
    try {
      const buildSystem = detectBuildSystem(ctx.projectRoot);
      if (buildSystem === "unknown") {
        ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: false, errorCategory: "NOT_SUPPORTED" });
        return err(notSupported("cannot determine build system (no pom.xml / build.gradle)"));
      }

      const { framework, dependencies } = detectTestFramework(ctx.projectRoot, buildSystem);
      const testRoots = detectTestRoots(ctx.projectRoot);
      const sourceRoots = DEFAULT_SOURCE_ROOTS.filter((s) => fileExists(ctx.projectRoot, s));

      const info: TestProjectInfo = {
        buildSystem,
        testFramework: framework,
        testRoots,
        sourceRoots,
        wrapperAvailable: detectWrapper(ctx.projectRoot),
        dependencies,
        existingTestPattern: detectExistingTestPattern(ctx.projectRoot, testRoots),
      };

      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: true });
      return ok(info);
    } catch (e) {
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: false, errorCategory: toToolError(e, "analyze_test_project failed").category });
      return err(toToolError(e, "analyze_test_project failed"));
    }
  },
};
