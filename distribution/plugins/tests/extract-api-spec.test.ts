import { describe, expect, test } from "bun:test";
import * as path from "node:path";
import {
  parseClassInfo,
  parseEndpoints,
  parseFieldDeclarations,
  parseEnumValues,
  extractApiSpecTool,
} from "../src/tools/extract-api-spec";
import { makeContext } from "./helpers";

const FIXTURE = path.resolve(import.meta.dir, "../../../tui/tests/e2e/fixtures/java-maven-project");

describe("parseClassInfo", () => {
  test("extracts controller name and base path", () => {
    const { controllerName, basePath } = parseClassInfo(
      '@RestController\n@RequestMapping("/api/users")\npublic class DemoController {}',
    );
    expect(controllerName).toBe("DemoController");
    expect(basePath).toBe("/api/users");
  });
});

describe("parseFieldDeclarations", () => {
  test("extracts DTO fields with validation", () => {
    const fields = parseFieldDeclarations(
      'public class X {\n  @NotBlank private String name;\n  @Min(1) @Max(120) private Integer age;\n  private String email;\n}',
    );
    expect(fields.length).toBe(3);
    expect(fields[0].name).toBe("name");
    expect(fields[0].validation).toContain("@NotBlank");
    expect(fields[1].validation).toContain("@Min(1)");
    expect(fields[1].validation).toContain("@Max(120)");
  });
});

describe("parseEnumValues", () => {
  test("extracts enum constants", () => {
    expect(parseEnumValues("public enum S { ACTIVE, INACTIVE, SUSPENDED }")).toEqual(["ACTIVE", "INACTIVE", "SUSPENDED"]);
  });
});

describe("extractApiSpecTool.execute (fixture)", () => {
  test("produces a deterministic spec with endpoints, DTOs and enums", async () => {
    const ctx = makeContext(FIXTURE).ctx;
    const result = await extractApiSpecTool.execute(
      { controllerFile: "src/main/java/com/example/demo/DemoController.java" },
      ctx,
    );
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data.controllerName).toBe("DemoController");
      expect(result.data.basePath).toBe("/api/users");
      expect(result.data.endpoints.length).toBeGreaterThanOrEqual(4);

      const get = result.data.endpoints.find((e) => e.method === "GET" && e.path === "/{id}");
      expect(get).toBeDefined();
      expect(get?.parameters.some((p) => p.location === "path")).toBe(true);

      const post = result.data.endpoints.find((e) => e.method === "POST");
      expect(post?.requestBody?.type).toBe("CreateUserRequest");

      expect(result.data.enums["UserStatus"]?.values).toEqual(["ACTIVE", "INACTIVE", "SUSPENDED"]);
      expect(result.data.dtos["CreateUserRequest"]).toBeDefined();
      expect(result.data.dtos["UserDto"]).toBeDefined();
    }
  });

  test("rejects an out-of-root controller path", async () => {
    const ctx = makeContext(FIXTURE).ctx;
    const result = await extractApiSpecTool.execute({ controllerFile: "../secret/Controller.java" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("PATH_VIOLATION");
  });
});
