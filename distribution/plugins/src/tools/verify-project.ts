import * as fs from "node:fs";
import * as path from "node:path";
import { scanDlp } from "../security/dlp";
import { execCommand, type ExecResult } from "./exec";
import { invalidInput } from "./errors";
import { validateSchema, type JsonSchema, type ValidationIssue } from "./schemas";
import { err, ok, type ToolContext, type ToolResult } from "./types";

export type VerificationCategory = "PASS" | "FAIL" | "TIMEOUT" | "NOT_CONFIGURED" | "ERROR";
export type VerificationProfileKind = "maven" | "gradle" | "go" | "unknown";

export interface VerificationStage {
  name: string;
  category: VerificationCategory;
  exitCode: number | null;
  durationMs: number;
  commandSummary: string;
  outputSummary: string;
}

export interface SkippedVerificationStage {
  name: string;
  reason: "PRIOR_STAGE_NOT_PASS";
}

export interface VerificationEvidence {
  profile: VerificationProfileKind;
  startedAt: string;
  finishedAt: string;
  stages: VerificationStage[];
  skippedStages: SkippedVerificationStage[];
  result: VerificationCategory;
  reason?: string;
}

export interface VerifyProjectInput {
  timeoutSeconds?: number;
}

export interface PlannedVerificationStage {
  name: string;
  argv: string[];
}

export interface VerificationProfile {
  kind: VerificationProfileKind;
  executable: string | null;
  stages: PlannedVerificationStage[];
  reason?: "AMBIGUOUS_BUILD_SYSTEM";
}

export type CommandRunner = (
  argv: readonly string[],
  opts: { cwd: string; timeoutMs?: number },
) => Promise<ExecResult>;

export const DEFAULT_STAGE_TIMEOUT_SECONDS = 180;
export const MAX_STAGE_TIMEOUT_SECONDS = 600;
export const MAX_OUTPUT_SUMMARY_CHARS = 2048;

const MAX_BUILD_FILE_BYTES = 256 * 1024;
const MAVEN_STATIC_MARKERS = ["maven-checkstyle-plugin", "maven-pmd-plugin", "spotbugs-maven-plugin"];
const GRADLE_STATIC_MARKERS = ["checkstyle", "pmd", "com.github.spotbugs"];

export const VERIFY_PROJECT_SCHEMA: JsonSchema = {
  type: "object",
  properties: {
    timeoutSeconds: { type: "integer", minimum: 1, maximum: MAX_STAGE_TIMEOUT_SECONDS },
  },
  additionalProperties: false,
};

export function validateVerifyProjectInput(value: unknown): ValidationIssue[] {
  return validateSchema(VERIFY_PROJECT_SCHEMA, value);
}

export function aggregateVerificationResult(
  profile: VerificationProfileKind,
  stages: Array<Pick<VerificationStage, "category">>,
): VerificationCategory {
  if (profile === "unknown") return "NOT_CONFIGURED";
  if (stages.length === 0) return "ERROR";
  if (stages.some((stage) => stage.category === "ERROR")) return "ERROR";
  if (stages.some((stage) => stage.category === "TIMEOUT")) return "TIMEOUT";
  if (stages.some((stage) => stage.category === "NOT_CONFIGURED")) return "NOT_CONFIGURED";
  if (stages.some((stage) => stage.category === "FAIL")) return "FAIL";
  if (stages.every((stage) => stage.category === "PASS")) return "PASS";
  return "ERROR";
}

function rootFileExists(root: string, name: string): boolean {
  try {
    return fs.statSync(path.join(root, name)).isFile();
  } catch {
    return false;
  }
}

function readBoundedRootFile(root: string, names: readonly string[]): string {
  for (const name of names) {
    const candidate = path.join(root, name);
    try {
      const stat = fs.statSync(candidate);
      if (!stat.isFile()) continue;
      if (stat.size > MAX_BUILD_FILE_BYTES) return "";
      return fs.readFileSync(candidate, "utf8");
    } catch {
      // Missing/unreadable optional build evidence contributes no marker evidence.
    }
  }
  return "";
}

function chooseWrapper(
  root: string,
  platform: string,
  unixName: string,
  windowsName: string,
  bare: string,
): string {
  if (platform === "win32" && rootFileExists(root, windowsName)) return `./${windowsName}`;
  if (platform !== "win32" && rootFileExists(root, unixName)) return `./${unixName}`;
  if (rootFileExists(root, platform === "win32" ? unixName : windowsName)) {
    return `./${platform === "win32" ? unixName : windowsName}`;
  }
  return bare;
}

function containsConfiguredMarker(content: string, markers: readonly string[]): boolean {
  return markers.some((marker) => content.includes(marker));
}

export function detectVerificationProfile(root: string, platform: string = process.platform): VerificationProfile {
  const hasMaven = rootFileExists(root, "pom.xml");
  const hasGradle = rootFileExists(root, "build.gradle") || rootFileExists(root, "build.gradle.kts");
  const hasGo = rootFileExists(root, "go.mod");

  if (hasMaven && hasGradle) {
    return { kind: "unknown", executable: null, stages: [], reason: "AMBIGUOUS_BUILD_SYSTEM" };
  }

  if (hasMaven) {
    const executable = chooseWrapper(root, platform, "mvnw", "mvnw.cmd", "mvn");
    const pom = readBoundedRootFile(root, ["pom.xml"]);
    const stages: PlannedVerificationStage[] = [
      { name: "compile", argv: [executable, "-DskipTests", "compile"] },
      { name: "test", argv: [executable, "test"] },
    ];
    if (containsConfiguredMarker(pom, MAVEN_STATIC_MARKERS)) {
      stages.push({ name: "verify", argv: [executable, "verify"] });
    }
    return { kind: "maven", executable, stages };
  }

  if (hasGradle) {
    const executable = chooseWrapper(root, platform, "gradlew", "gradlew.bat", "gradle");
    const build = readBoundedRootFile(root, ["build.gradle", "build.gradle.kts"]);
    const stages: PlannedVerificationStage[] = [
      { name: "classes", argv: [executable, "classes"] },
      { name: "test", argv: [executable, "test"] },
    ];
    if (containsConfiguredMarker(build, GRADLE_STATIC_MARKERS)) {
      stages.push({ name: "check", argv: [executable, "check"] });
    }
    return { kind: "gradle", executable, stages };
  }

  if (hasGo) {
    return {
      kind: "go",
      executable: "go",
      stages: [
        { name: "test", argv: ["go", "test", "./..."] },
        { name: "vet", argv: ["go", "vet", "./..."] },
      ],
    };
  }

  return { kind: "unknown", executable: null, stages: [] };
}

function summarizeOutput(value: string): string {
  const normalized = value.replace(/\r\n/g, "\n").replace(/\r/g, "\n").trim();
  const redacted = scanDlp(normalized, "tool-output").redacted;
  if (redacted.length <= MAX_OUTPUT_SUMMARY_CHARS) return redacted;
  return redacted.slice(0, MAX_OUTPUT_SUMMARY_CHARS);
}

function categoryForResult(result: ExecResult): VerificationCategory {
  if (result.timedOut) return "TIMEOUT";
  if (result.exitCode === 0) return "PASS";
  if (typeof result.exitCode === "number") return "FAIL";
  return "ERROR";
}

function evidenceOutput(result: ExecResult): string {
  const combined = [result.stdout, result.stderr].filter((part) => part !== "").join("\n");
  return summarizeOutput(combined);
}

export function createVerifyProjectTool(runner: CommandRunner = execCommand) {
  return {
    name: "verify_project",
    description: "Run deterministic local verification for the detected Maven, Gradle, or Go project profile.",
    async execute(params: unknown, ctx: ToolContext): Promise<ToolResult<VerificationEvidence>> {
      const input = params && typeof params === "object" ? params : {};
      const issues = validateVerifyProjectInput(input);
      if (issues.length > 0) {
        return err(invalidInput(issues.map((issue) => `${issue.path}: ${issue.message}`).join("; ")));
      }

      const startedAt = new Date().toISOString();
      const profile = detectVerificationProfile(ctx.projectRoot);
      if (profile.kind === "unknown") {
        const evidence: VerificationEvidence = {
          profile: "unknown",
          startedAt,
          finishedAt: new Date().toISOString(),
          stages: [],
          skippedStages: [],
          result: "NOT_CONFIGURED",
          ...(profile.reason ? { reason: profile.reason } : {}),
        };
        ctx.guard.after({
          sessionId: ctx.sessionId, agent: ctx.agent, tool: "verify_project", action: "execute",
          projectRoot: ctx.projectRoot, durationMs: 0, ok: true, errorCategory: evidence.result, output: evidence,
        });
        return ok(evidence);
      }

      const timeoutSeconds = (input as VerifyProjectInput).timeoutSeconds ?? DEFAULT_STAGE_TIMEOUT_SECONDS;
      const stages: VerificationStage[] = [];
      const skippedStages: SkippedVerificationStage[] = [];
      const overallStart = Date.now();

      for (let index = 0; index < profile.stages.length; index += 1) {
        const planned = profile.stages[index]!;
        if (stages.some((stage) => stage.category !== "PASS")) {
          skippedStages.push({ name: planned.name, reason: "PRIOR_STAGE_NOT_PASS" });
          continue;
        }

        const stageStart = Date.now();
        try {
          const execution = await runner(planned.argv, { cwd: ctx.projectRoot, timeoutMs: timeoutSeconds * 1000 });
          stages.push({
            name: planned.name,
            category: categoryForResult(execution),
            exitCode: execution.exitCode,
            durationMs: Math.max(0, Date.now() - stageStart),
            commandSummary: planned.argv.join(" "),
            outputSummary: evidenceOutput(execution),
          });
        } catch (error) {
          stages.push({
            name: planned.name,
            category: "ERROR",
            exitCode: null,
            durationMs: Math.max(0, Date.now() - stageStart),
            commandSummary: planned.argv.join(" "),
            outputSummary: summarizeOutput(error instanceof Error ? error.message : String(error)),
          });
        }
      }

      const evidence: VerificationEvidence = {
        profile: profile.kind,
        startedAt,
        finishedAt: new Date().toISOString(),
        stages,
        skippedStages,
        result: aggregateVerificationResult(profile.kind, stages),
      };
      ctx.guard.after({
        sessionId: ctx.sessionId, agent: ctx.agent, tool: "verify_project", action: "execute",
        projectRoot: ctx.projectRoot, durationMs: Math.max(0, Date.now() - overallStart),
        ok: true, errorCategory: evidence.result, output: evidence,
      });
      return ok(evidence);
    },
  };
}

export const verifyProjectTool = createVerifyProjectTool();
