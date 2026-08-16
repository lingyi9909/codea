import { describe, expect, test } from "bun:test";
import { validateSchema, isValid, type JsonSchema } from "../src/tools/schemas";

const SCHEMA: JsonSchema = {
  type: "object",
  properties: {
    source: { type: "string", enum: ["staged", "unstaged"] },
    count: { type: "integer", minimum: 1, maximum: 10 },
    tags: { type: "array", items: { type: "string" } },
  },
  required: ["source"],
  additionalProperties: false,
};

describe("validateSchema", () => {
  test("accepts a valid object", () => {
    expect(isValid(SCHEMA, { source: "staged", count: 5, tags: ["a"] })).toBe(true);
  });

  test("rejects missing required field", () => {
    const issues = validateSchema(SCHEMA, { count: 5 });
    expect(issues.some((i) => i.message === "required")).toBe(true);
  });

  test("rejects invalid enum", () => {
    const issues = validateSchema(SCHEMA, { source: "nope" });
    expect(issues.some((i) => i.message.includes("must be one of"))).toBe(true);
  });

  test("rejects out-of-range integer", () => {
    const issues = validateSchema(SCHEMA, { source: "staged", count: 99 });
    expect(issues.some((i) => i.message.includes("<= 10"))).toBe(true);
  });

  test("rejects wrong type", () => {
    const issues = validateSchema(SCHEMA, { source: "staged", count: "not-a-number" });
    expect(issues.some((i) => i.message.includes("expected integer"))).toBe(true);
  });

  test("rejects unknown field when additionalProperties=false", () => {
    const issues = validateSchema(SCHEMA, { source: "staged", evil: true });
    expect(issues.some((i) => i.message === "not allowed")).toBe(true);
  });

  test("rejects non-object root", () => {
    const issues = validateSchema(SCHEMA, "not-an-object");
    expect(issues.some((i) => i.message === "expected object")).toBe(true);
  });

  test("validates array item types", () => {
    const issues = validateSchema(SCHEMA, { source: "staged", tags: [1, 2, 3] });
    expect(issues.some((i) => i.message.includes("expected string"))).toBe(true);
  });
});
