import { describe, expect, test } from "bun:test";

const verificationModule = await import("../src/tools/verify-project").catch(() => ({} as Record<string, unknown>));

type ValidateInput = (value: unknown) => Array<{ path: string; message: string }>;
type Aggregate = (
  profile: "maven" | "gradle" | "go" | "unknown",
  stages: Array<{ category: "PASS" | "FAIL" | "TIMEOUT" | "NOT_CONFIGURED" | "ERROR" }>,
) => "PASS" | "FAIL" | "TIMEOUT" | "NOT_CONFIGURED" | "ERROR";

function validateInput(): ValidateInput {
  expect(typeof verificationModule.validateVerifyProjectInput).toBe("function");
  return verificationModule.validateVerifyProjectInput as ValidateInput;
}

function aggregate(): Aggregate {
  expect(typeof verificationModule.aggregateVerificationResult).toBe("function");
  return verificationModule.aggregateVerificationResult as Aggregate;
}

describe("verify_project input contract", () => {
  test("accepts empty input and bounded timeout only", () => {
    const validate = validateInput();
    expect(validate({})).toEqual([]);
    expect(validate({ timeoutSeconds: 1 })).toEqual([]);
    expect(validate({ timeoutSeconds: 180 })).toEqual([]);
    expect(validate({ timeoutSeconds: 600 })).toEqual([]);
  });

  test("rejects arbitrary command and args fields", () => {
    const validate = validateInput();
    expect(validate({ command: "mvn test" }).map((issue) => issue.path)).toContain("$.command");
    expect(validate({ args: ["test"] }).map((issue) => issue.path)).toContain("$.args");
    expect(validate({ repository: "https://example.invalid" }).map((issue) => issue.path)).toContain("$.repository");
  });

  test("rejects invalid timeout values", () => {
    const validate = validateInput();
    expect(validate({ timeoutSeconds: 0 }).length).toBeGreaterThan(0);
    expect(validate({ timeoutSeconds: -1 }).length).toBeGreaterThan(0);
    expect(validate({ timeoutSeconds: 601 }).length).toBeGreaterThan(0);
    expect(validate({ timeoutSeconds: 1.5 }).length).toBeGreaterThan(0);
  });
});

describe("verify_project deterministic result aggregation", () => {
  test("returns PASS only when configured executed stages all pass", () => {
    const result = aggregate();
    expect(result("go", [{ category: "PASS" }, { category: "PASS" }])).toBe("PASS");
    expect(result("maven", [])).toBe("ERROR");
  });

  test("never converts unknown profile to PASS", () => {
    const result = aggregate();
    expect(result("unknown", [])).toBe("NOT_CONFIGURED");
    expect(result("unknown", [{ category: "PASS" }])).toBe("NOT_CONFIGURED");
  });

  test("uses deterministic failure precedence", () => {
    const result = aggregate();
    expect(result("gradle", [{ category: "PASS" }, { category: "FAIL" }])).toBe("FAIL");
    expect(result("gradle", [{ category: "FAIL" }, { category: "TIMEOUT" }])).toBe("TIMEOUT");
    expect(result("gradle", [{ category: "TIMEOUT" }, { category: "ERROR" }])).toBe("ERROR");
  });
});
