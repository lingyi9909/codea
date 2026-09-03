import { fileExists } from "./filesystem";
import { execCommand } from "./exec";
import { invalidInput } from "./errors";
import { toToolError } from "./failure-classifier";
import { validateSchema, type JsonSchema } from "./schemas";
import { err, ok, type ToolContext, type ToolResult } from "./types";

// Unit Test execution tool. Prefers the host-native Maven/Gradle wrapper,
// falls back to the alternate wrapper and then to bare mvn/gradle. Always argv
// arrays (no arbitrary shell), no caller-supplied extra args (no Maven/Gradle
// extension bypass). Output is parsed into structured pass/fail counts.

export type TestRunCategory = "PASS" | "FAIL" | "TIMEOUT" | "ERROR";

export interface RunProjectTestInput {
  buildSystem: "maven" | "gradle";
  module?: string;
  testClass?: string;
  testMethod?: string;
  profiles?: string[];
  timeoutSeconds?: number;
}

export interface TestRunResult {
  passed: number;
  failed: number;
  errors: number;
  skipped: number;
  duration: number;
  failureDetails: string[];
  exitCode: number;
  category: TestRunCategory;
}

const DEFAULT_TIMEOUT_SECONDS = 120;
const MAX_TIMEOUT_SECONDS = 600;

const SCHEMA: JsonSchema = {
  type: "object",
  properties: {
    buildSystem: { type: "string", enum: ["maven", "gradle"] },
    module: { type: "string" },
    testClass: { type: "string" },
    testMethod: { type: "string" },
    profiles: { type: "array", items: { type: "string" } },
    timeoutSeconds: { type: "integer", minimum: 1 },
  },
  required: ["buildSystem"],
  additionalProperties: false,
};

const WRAPPERS: Record<"maven" | "gradle", { posix: string; windows: string }> = {
  maven: { posix: "mvnw", windows: "mvnw.cmd" },
  gradle: { posix: "gradlew", windows: "gradlew.bat" },
};

// Shell/cmd metacharacters forbidden in caller-supplied build args. On Unix the
// argv is passed to execFile (no shell), but on Windows the .cmd/.bat wrapper is
// routed through `cmd.exe /c`, where these become live. Rejecting them up front
// keeps the batch path injection-free for every platform.
const UNSAFE_BUILD_ARG = /[\s&|<>^%!"'`();]/;

function assertSafeBuildArgs(input: RunProjectTestInput): void {
  for (const field of ["module", "testClass", "testMethod"] as const) {
    const v = input[field];
    if (typeof v === "string" && UNSAFE_BUILD_ARG.test(v)) {
      throw invalidInput(`unsafe characters in ${field}`);
    }
  }
  if (input.profiles) {
    for (const p of input.profiles) {
      if (UNSAFE_BUILD_ARG.test(p)) throw invalidInput("unsafe characters in profiles");
    }
  }
}

function detectWrapper(root: string, buildSystem: "maven" | "gradle", platform: string = process.platform): string | null {
  const pair = WRAPPERS[buildSystem];
  const candidates = platform === "win32" ? [pair.windows, pair.posix] : [pair.posix, pair.windows];
  for (const name of candidates) {
    if (fileExists(root, name)) return name;
  }
  return null;
}

export function buildCommand(input: RunProjectTestInput, root: string, platform: string = process.platform): string[] {
  const isMaven = input.buildSystem === "maven";
  const bare = isMaven ? "mvn" : "gradle";
  const wrapper = detectWrapper(root, isMaven ? "maven" : "gradle", platform);
  const base = wrapper ? `./${wrapper}` : bare;

  const argv: string[] = [base];

  if (isMaven) {
    if (input.profiles && input.profiles.length > 0) argv.push(`-P${input.profiles.join(",")}`);
    if (input.module) argv.push("-pl", input.module);
    const testSelector = input.testMethod ? `${input.testClass}#${input.testMethod}` : input.testClass;
    if (testSelector) argv.push(`-Dtest=${testSelector}`);
    argv.push("test");
  } else {
    if (input.module) argv.push(`${input.module}:test`);
    else argv.push("test");
    if (input.testClass) argv.push("--tests", input.testMethod ? `${input.testClass}.${input.testMethod}` : input.testClass);
  }

  return argv;
}

// Parses Maven Surefire "Tests run: X, Failures: Y, Errors: Z, Skipped: W" and
// Gradle "X tests completed, Y failed" summaries. Deterministic; missing counts
// default to 0 rather than being guessed.
export function parseTestSummary(stdout: string): { passed: number; failed: number; errors: number; skipped: number; failureDetails: string[] } {
  let passed = 0;
  let failed = 0;
  let errors = 0;
  let skipped = 0;

  const surefire = /Tests run:\s*(\d+),\s*Failures:\s*(\d+),\s*Errors:\s*(\d+),\s*Skipped:\s*(\d+)/.exec(stdout);
  if (surefire) {
    const total = parseInt(surefire[1] ?? "0", 10);
    failed = parseInt(surefire[2] ?? "0", 10);
    errors = parseInt(surefire[3] ?? "0", 10);
    skipped = parseInt(surefire[4] ?? "0", 10);
    passed = total - failed - errors - skipped;
  } else {
    const gradle = /(\d+)\s+tests?\s+completed,\s+(\d+)\s+failed/.exec(stdout);
    if (gradle) {
      const total = parseInt(gradle[1] ?? "0", 10);
      failed = parseInt(gradle[2] ?? "0", 10);
      passed = total - failed;
    }
  }

  const failureDetails = stdout
    .split("\n")
    .filter((l) => /\[ERROR\]|FAILED|<<< FAILURE|AssertionError|BUILD FAILURE/.test(l))
    .slice(0, 20);

  return { passed, failed, errors, skipped, failureDetails };
}

export const runProjectTestTool = {
  name: "run_project_test",
  description: "Run project tests using Maven or Gradle wrapper.",
  parameters: SCHEMA,

  async execute(params: unknown, ctx: ToolContext): Promise<ToolResult<TestRunResult>> {
    const started = Date.now();
    try {
      const issues = validateSchema(SCHEMA, params);
      if (issues.length > 0) {
        throw invalidInput(`invalid input: ${issues.map((i) => `${i.path} ${i.message}`).join("; ")}`);
      }
      const input = params as RunProjectTestInput;
      assertSafeBuildArgs(input);

      const timeoutMs = Math.min(input.timeoutSeconds ?? DEFAULT_TIMEOUT_SECONDS, MAX_TIMEOUT_SECONDS) * 1000;
      const argv = buildCommand(input, ctx.projectRoot);
      const result = await execCommand(argv, { cwd: ctx.projectRoot, timeoutMs });

      const summary = parseTestSummary(result.stdout);

      let category: TestRunCategory;
      if (result.timedOut) category = "TIMEOUT";
      else if (result.exitCode === 0) category = summary.failed > 0 || summary.errors > 0 ? "FAIL" : "PASS";
      else category = "FAIL";

      const output: TestRunResult = {
        passed: summary.passed,
        failed: summary.failed,
        errors: summary.errors,
        skipped: summary.skipped,
        duration: (Date.now() - started) / 1000,
        failureDetails: summary.failureDetails,
        exitCode: result.exitCode ?? 1,
        category,
      };

      const okResult = category === "PASS";
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "execute", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: okResult, errorCategory: okResult ? undefined : "COMMAND_FAILED" });
      return ok(output);
    } catch (e) {
      const toolErr = toToolError(e, "run_project_test failed");
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "execute", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: false, errorCategory: toolErr.category });
      return err(toolErr);
    }
  },
};
