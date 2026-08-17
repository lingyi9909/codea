import { describe, expect, test } from "bun:test";
import { validateExample, validateApiExampleTool } from "../src/tools/validate-api-example";
import type { ApiSpecOutput } from "../src/tools/extract-api-spec";
import { makeTempRoot, makeContext } from "./helpers";

const SPEC: ApiSpecOutput = {
  controllerName: "DemoController",
  basePath: "/api/users",
  endpoints: [
    {
      method: "POST",
      path: "",
      summary: "",
      parameters: [],
      requestBody: { type: "CreateUserRequest", fields: [] },
      responseType: "UserDto",
      errorCodes: [],
    },
  ],
  dtos: {
    CreateUserRequest: {
      fields: [
        { name: "name", type: "String", validation: ["@NotBlank"], description: "" },
        { name: "age", type: "Integer", validation: ["@Min(1)", "@Max(120)"], description: "" },
        { name: "status", type: "UserStatus", validation: [], description: "" },
      ],
    },
  },
  enums: { UserStatus: { values: ["ACTIVE", "INACTIVE", "SUSPENDED"] } },
};

describe("validateExample", () => {
  test("accepts a valid example", () => {
    const r = validateExample({ example: { name: "Alice", age: 30, status: "ACTIVE" }, spec: SPEC, endpointIndex: 0 });
    expect(r.valid).toBe(true);
    expect(r.errors).toEqual([]);
  });

  test("rejects a missing required field", () => {
    const r = validateExample({ example: { age: 30 }, spec: SPEC, endpointIndex: 0 });
    expect(r.valid).toBe(false);
    expect(r.errors.some((e) => e.includes('"name"'))).toBe(true);
  });

  test("rejects an unknown field (not a warning)", () => {
    const r = validateExample({ example: { name: "A", age: 30, hack: true }, spec: SPEC, endpointIndex: 0 });
    expect(r.valid).toBe(false);
    expect(r.errors.some((e) => e.includes('"hack"'))).toBe(true);
  });

  test("rejects an out-of-enum value", () => {
    const r = validateExample({ example: { name: "A", age: 30, status: "BANNED" }, spec: SPEC, endpointIndex: 0 });
    expect(r.valid).toBe(false);
    expect(r.errors.some((e) => e.includes("enum"))).toBe(true);
  });

  test("rejects an out-of-range number", () => {
    const r = validateExample({ example: { name: "A", age: 999, status: "ACTIVE" }, spec: SPEC, endpointIndex: 0 });
    expect(r.valid).toBe(false);
    expect(r.errors.some((e) => e.includes("@Max"))).toBe(true);
  });
});

describe("validateApiExampleTool.execute", () => {
  test("returns valid for a correct example", async () => {
    const root = makeTempRoot("codea-validate-");
    const { ctx } = makeContext(root);
    const result = await validateApiExampleTool.execute(
      { example: { name: "A", age: 30, status: "ACTIVE" }, spec: SPEC, endpointIndex: 0 },
      ctx,
    );
    expect(result.ok).toBe(true);
    if (result.ok) expect(result.data.valid).toBe(true);
  });

  test("rejects out-of-range endpointIndex", async () => {
    const root = makeTempRoot("codea-validate-");
    const { ctx } = makeContext(root);
    const result = await validateApiExampleTool.execute(
      { example: {}, spec: SPEC, endpointIndex: 5 },
      ctx,
    );
    expect(result.ok).toBe(true);
    if (result.ok) expect(result.data.valid).toBe(false);
  });
});
