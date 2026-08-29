import * as fs from "node:fs";
import * as path from "node:path";
import { validateSchema, type JsonSchema, type ValidationIssue } from "./schemas";

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

export interface VerificationEvidence {
  profile: VerificationProfileKind;
  startedAt: string;
  finishedAt: string;
  stages: VerificationStage[];
  result: VerificationCategory;
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

export const DEFAULT_STAGE_TIMEOUT_SECONDS = 180;
export const MAX_STAGE_TIMEOUT_SECONDS = 600;

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
      // A missing/unreadable optional build file simply provides no local marker evidence.
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
