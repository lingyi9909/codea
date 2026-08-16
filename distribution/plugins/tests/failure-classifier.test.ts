import { describe, expect, test } from "bun:test";
import { classifyError, toToolError, classifyCommandFailure } from "../src/tools/failure-classifier";
import { ToolError } from "../src/tools/errors";
import { PathViolationError } from "../src/security/path-policy";

describe("classifyError", () => {
  test("preserves a ToolError category", () => {
    const e = new ToolError("DLP_BLOCKED", "secret found");
    expect(classifyError(e)).toBe("DLP_BLOCKED");
  });

  test("maps PathViolationError to PATH_VIOLATION", () => {
    expect(classifyError(new PathViolationError("escape"))).toBe("PATH_VIOLATION");
  });

  test("maps killed process to TIMEOUT", () => {
    const e = Object.assign(new Error("killed"), { killed: true });
    expect(classifyError(e)).toBe("TIMEOUT");
  });

  test("maps ETIMEDOUT to TIMEOUT", () => {
    const e = Object.assign(new Error("timeout"), { code: "ETIMEDOUT" });
    expect(classifyError(e)).toBe("TIMEOUT");
  });

  test("maps unknown error to INTERNAL_ERROR", () => {
    expect(classifyError(new Error("boom"))).toBe("INTERNAL_ERROR");
    expect(classifyError("a string")).toBe("INTERNAL_ERROR");
  });
});

describe("toToolError", () => {
  test("returns existing ToolError unchanged", () => {
    const e = new ToolError("PARSE_FAILED", "x");
    expect(toToolError(e, "fallback")).toBe(e);
  });

  test("wraps a plain Error with INTERNAL_ERROR", () => {
    const e = toToolError(new Error("boom"), "fallback");
    expect(e.category).toBe("INTERNAL_ERROR");
    expect(e.message).toBe("boom");
  });
});

describe("classifyCommandFailure", () => {
  test("timeout", () => {
    expect(classifyCommandFailure(null, true)).toBe("TIMEOUT");
  });
  test("nonzero exit", () => {
    expect(classifyCommandFailure(1, false)).toBe("COMMAND_FAILED");
  });
});
