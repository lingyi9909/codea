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

export const DEFAULT_STAGE_TIMEOUT_SECONDS = 180;
export const MAX_STAGE_TIMEOUT_SECONDS = 600;

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
