import { describe, expect, test } from "bun:test";
import * as path from "node:path";
import {
  parseClassInfo,
  parseEndpoints,
  parseFieldDeclarations,
  parseEnumValues,
  extractApiSpecTool,
  joinPaths,
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

describe("joinPaths", () => {
  test("merges base path and endpoint path", () => {
    expect(joinPaths("/api/users", "/{id}")).toBe("/api/users/{id}");
  });
  test("empty endpoint inherits base path", () => {
    expect(joinPaths("/api/users", "")).toBe("/api/users");
  });
  test("empty base returns endpoint", () => {
    expect(joinPaths("", "/{id}")).toBe("/{id}");
  });
  test("root endpoint path keeps base", () => {
    expect(joinPaths("/api/users", "/")).toBe("/api/users");
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

describe("parseEndpoints — generic params and method-scoped error codes", () => {
  test("does not split generic type parameters", () => {
    const src = [
      "@RestController",
      "public class C {",
      '  @PostMapping("/search")',
      "  public Map<String, Object> search(@RequestBody Map<String, Object> body) {",
      "    return Map.of();",
      "  }",
      "}",
    ].join("\n");
    const eps = parseEndpoints(src);
    expect(eps.length).toBe(1);
    expect(eps[0].requestBody?.type).toContain("Map");
  });

  test("throw in one endpoint does not leak into another", () => {
    const src = [
      "@RestController",
      "public class C {",
      '  @GetMapping("/a")',
      "  public void a() { throw new NotFoundException(); }",
      '  @GetMapping("/b")',
      "  public void b() { }",
      "}",
    ].join("\n");
    const eps = parseEndpoints(src);
    const a = eps.find((e) => e.path === "/a");
    const b = eps.find((e) => e.path === "/b");
    expect(a?.errorCodes.some((c) => c.code === "NotFoundException")).toBe(true);
    expect(b?.errorCodes.some((c) => c.code === "NotFoundException")).toBe(false);
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

      const get = result.data.endpoints.find((e) => e.method === "GET" && e.path === "/api/users/{id}");
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
