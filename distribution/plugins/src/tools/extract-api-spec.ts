import * as fs from "node:fs";
import * as path from "node:path";
import { resolveInRoot } from "./filesystem";
import { invalidInput } from "./errors";
import { toToolError } from "./failure-classifier";
import { validateSchema, type JsonSchema } from "./schemas";
import { err, ok, type ToolContext, type ToolResult } from "./types";

// API Doc tool: deterministically extracts a Spring MVC controller's routes,
// parameters, DTOs and enums into a structured spec. V1 is a finite-syntax
// parser, not a Java compiler — anything it cannot determine is marked
// "Not determined from code" rather than fabricated.

export interface FieldInfo {
  name: string;
  type: string;
  validation: string[];
  description: string;
}

export interface ApiParameter {
  name: string;
  type: string;
  required: boolean;
  location: "path" | "query" | "body" | "header";
  validation: string[];
  description: string;
}

export interface ApiErrorCode {
  code: string;
  status: string;
  source: "DECLARED" | "REFERENCED" | "INFERRED";
}

export interface ApiEndpoint {
  method: string;
  path: string;
  summary: string;
  parameters: ApiParameter[];
  requestBody?: { type: string; fields: FieldInfo[] };
  responseType: string;
  errorCodes: ApiErrorCode[];
}

export interface ApiSpecOutput {
  controllerName: string;
  basePath: string;
  endpoints: ApiEndpoint[];
  dtos: Record<string, { fields: FieldInfo[] }>;
  enums: Record<string, { values: string[] }>;
}

export interface ExtractApiSpecInput {
  controllerFile: string;
}

const NOT_DETERMINED = "Not determined from code";

const SCHEMA: JsonSchema = {
  type: "object",
  properties: {
    controllerFile: { type: "string", minLength: 1 },
  },
  required: ["controllerFile"],
  additionalProperties: false,
};

const HTTP_METHOD_ANNOTATIONS: Array<[string, RegExp]> = [
  ["GET", /@GetMapping/],
  ["POST", /@PostMapping/],
  ["PUT", /@PutMapping/],
  ["DELETE", /@DeleteMapping/],
  ["PATCH", /@PatchMapping/],
];

function extractStringArg(annotation: string): string | undefined {
  const m = /(?:value|path)\s*=\s*"([^"]+)"/.exec(annotation) || /@\w+Mapping\s*\(\s*"([^"]+)"/.exec(annotation) || /@\w+Mapping\s*\(\s*"([^"]+)"/.exec(annotation);
  return m?.[1];
}

// Merge a class-level base path with an endpoint path: "/api/users" + "/{id}"
// -> "/api/users/{id}". Empty endpoint paths inherit the base path.
export function joinPaths(basePath: string, endpointPath: string): string {
  const base = basePath || "";
  const endpoint = endpointPath || "";
  if (!base) return endpoint || "/";
  if (!endpoint || endpoint === "/") return base;
  const b = base.endsWith("/") ? base.slice(0, -1) : base;
  const e = endpoint.startsWith("/") ? endpoint : `/${endpoint}`;
  return `${b}${e}`;
}

// Split on commas at the top level only, so generic type arguments such as
// `Map<String, Object>` do not break the parameter list.
function splitTopLevel(text: string): string[] {
  const parts: string[] = [];
  let depth = 0;
  let current = "";
  for (const ch of text) {
    if (ch === "<") depth += 1;
    else if (ch === ">") depth = Math.max(0, depth - 1);
    if (ch === "," && depth === 0) {
      parts.push(current);
      current = "";
    } else {
      current += ch;
    }
  }
  if (current.trim()) parts.push(current);
  return parts;
}

// Returns the balanced `{ ... }` block starting at openIndex.
function balancedBlock(source: string, openIndex: number): string {
  let depth = 0;
  for (let i = openIndex; i < source.length; i++) {
    const ch = source[i];
    if (ch === "{") depth += 1;
    else if (ch === "}") {
      depth -= 1;
      if (depth === 0) return source.slice(openIndex, i + 1);
    }
  }
  return source.slice(openIndex);
}

function extractValidation(block: string): string[] {
  const out: string[] = [];
  if (/@NotNull\b/.test(block)) out.push("@NotNull");
  if (/@NotBlank\b/.test(block)) out.push("@NotBlank");
  if (/@NotEmpty\b/.test(block)) out.push("@NotEmpty");
  if (/@Email\b/.test(block)) out.push("@Email");
  const size = /@Size\([^)]*\)/.exec(block);
  if (size) out.push(size[0]);
  const min = /@Min\([^)]*\)/.exec(block);
  if (min) out.push(min[0]);
  const max = /@Max\([^)]*\)/.exec(block);
  if (max) out.push(max[0]);
  return out;
}

export function parseClassInfo(source: string): { controllerName: string; basePath: string } {
  const classMatch = /(?:public\s+)?(?:final\s+)?class\s+(\w+)/.exec(source);
  const controllerName = classMatch?.[1] ?? NOT_DETERMINED;

  const rm = /@RequestMapping\s*\(([^)]*)\)/.exec(source);
  let basePath = "";
  if (rm) {
    basePath = extractStringArg(rm[0]) ?? "";
    if (!basePath) {
      const v = /(?:value|path)\s*=\s*\{?\s*"([^"]+)"/.exec(rm[1] ?? "");
      basePath = v?.[1] ?? "";
    }
  }
  return { controllerName, basePath };
}

export function parseEndpoints(source: string): ApiEndpoint[] {
  const endpoints: ApiEndpoint[] = [];

  // Iterate annotation occurrences and read forward to the method signature.
  // The argument list is optional (e.g. `@PostMapping` with no parentheses).
  const annRe = /(@(?:Get|Post|Put|Delete|Patch|Request)Mapping)(?:\(([^)]*)\))?/g;
  let m: RegExpExecArray | null;
  while ((m = annRe.exec(source)) !== null) {
    const annName = m[1] ?? "";
    const annArgs = m[2] ?? "";
    const rest = source.slice(annRe.lastIndex);
    const sigMatch = /(?:public|private|protected)?\s*([\w<>\[\].,\s]+?)\s+(\w+)\s*\(([\s\S]*?)\)\s*(?:throws[^{]+)?\{/.exec(rest);
    if (!sigMatch) continue;

    const returnType = (sigMatch[1] ?? "").trim();
    const paramsText = sigMatch[3] ?? "";

    let method = "GET";
    let path = "";
    if (annName === "@RequestMapping") {
      // Class-level @RequestMapping carries no `method=`; skip it (it is a base
      // path, not an endpoint). Method-level ones select RequestMethod.X.
      const mm = /method\s*=\s*RequestMethod\.(\w+)/.exec(annArgs);
      if (!mm) continue;
      method = (mm[1] ?? "").toUpperCase();
      path = extractStringArg(annName + (annArgs ? `(${annArgs})` : "")) ?? "";
    } else {
      for (const [hm, re] of HTTP_METHOD_ANNOTATIONS) {
        if (re.test(annName)) {
          method = hm;
          break;
        }
      }
      path = extractStringArg(annName + (annArgs ? `(${annArgs})` : "")) ?? "";
    }

    const parameters = parseParameters(paramsText);

    // request body: a @RequestBody param references a DTO type
    const bodyParam = parameters.find((p) => p.location === "body");
    const requestBody = bodyParam ? { type: bodyParam.type, fields: [] as FieldInfo[] } : undefined;

    // Error codes: declared handlers are class-level, referenced throws are
    // scoped to this method's body only (never the remainder of the file).
    const openBrace = sigMatch.index + sigMatch[0].lastIndexOf("{");
    const methodBody = balancedBlock(rest, openBrace);
    const errorCodes = collectErrorCodes(source, methodBody);

    endpoints.push({
      method,
      path,
      summary: "",
      parameters,
      requestBody,
      responseType: returnType,
      errorCodes,
    });
  }

  return endpoints;
}

function parseParameters(paramsText: string): ApiParameter[] {
  if (!paramsText.trim()) return [];
  const parameters: ApiParameter[] = [];

  // Split on top-level commas only, so generic type arguments (Map<String,Object>)
  // are not treated as parameter separators.
  const parts = splitTopLevel(paramsText);
  for (const part of parts) {
    const p = part.trim();
    if (!p) continue;

    let location: ApiParameter["location"] = "query";
    if (/@PathVariable/.test(p)) location = "path";
    else if (/@RequestBody/.test(p)) location = "body";
    else if (/@RequestHeader/.test(p)) location = "header";
    else if (/@RequestParam/.test(p)) location = "query";

    const nameMatch = /(?:@\w+(?:\([^)]*\))?\s*)*[\w<>\[\],\s]+\s+(\w+)\s*$/.exec(p) || /(?:@\w+(?:\([^)]*\))?\s*)*(\w+)\s*$/.exec(p);
    const name = nameMatch?.[1] ?? NOT_DETERMINED;

    const typeMatch = /(?:@\w+(?:\([^)]*\))?\s*)+([\w<>\[\],\s]+?)\s+\w+\s*$/.exec(p) || /([\w<>\[\],]+)\s+\w+\s*$/.exec(p);
    const type = typeMatch?.[1]?.trim() ?? NOT_DETERMINED;

    const required = location === "path" || (location === "body" && !/@\w+\(required\s*=\s*false\)/.test(p)) || (!/@\w+\(required\s*=\s*false\)/.test(p) && location === "query" && /@RequestParam/.test(p) && !/required\s*=\s*false/.test(p));

    parameters.push({
      name,
      type,
      required,
      location,
      validation: extractValidation(p),
      description: "",
    });
  }

  return parameters;
}

function collectErrorCodes(classSource: string, methodBody: string): ApiErrorCode[] {
  const codes: ApiErrorCode[] = [];

  // DECLARED error handlers are class-level (@ExceptionHandler/@ResponseStatus)
  // and apply to every endpoint, regardless of their position in the file.
  const declaredRe = /@ExceptionHandler\s*\(\s*(\w+)\.class\s*\)/g;
  let m: RegExpExecArray | null;
  while ((m = declaredRe.exec(classSource)) !== null) {
    codes.push({ code: m[1] ?? "", status: "", source: "DECLARED" });
  }
  const statusRe = /@ResponseStatus\s*\(\s*(?:code\s*=\s*)?(?:HttpStatus\.)?(\w+)\s*\)/g;
  while ((m = statusRe.exec(classSource)) !== null) {
    codes.push({ code: m[1] ?? "", status: m[1] ?? "", source: "DECLARED" });
  }

  // REFERENCED error codes come from `throw new XxxException` and are scoped to
  // the method body, so a throw in one endpoint never leaks into another.
  const throwRe = /throw\s+new\s+(\w+Exception)\s*\(/g;
  while ((m = throwRe.exec(methodBody)) !== null) {
    codes.push({ code: m[1] ?? "", status: "", source: "REFERENCED" });
  }

  if (codes.length === 0) {
    codes.push({ code: "500", status: "INTERNAL_SERVER_ERROR", source: "INFERRED" });
  }

  // de-dup by code+source
  const seen = new Set<string>();
  return codes.filter((c) => {
    const k = `${c.code}:${c.source}`;
    if (seen.has(k)) return false;
    seen.add(k);
    return true;
  });
}

export function parseFieldDeclarations(source: string): FieldInfo[] {
  const fields: FieldInfo[] = [];
  const re = /(@(?:NotNull|NotBlank|NotEmpty|Email|Size|Min|Max|Pattern)(?:\([^)]*\))?\s*)*\s*(?:private|protected|public)\s+([\w<>\[\],\s]+?)\s+(\w+)\s*;/g;
  let m: RegExpExecArray | null;
  while ((m = re.exec(source)) !== null) {
    const validationBlock = m[0];
    fields.push({
      name: m[3] ?? "",
      type: (m[2] ?? "").trim(),
      validation: extractValidation(validationBlock),
      description: "",
    });
  }
  return fields;
}

export function parseEnumValues(source: string): string[] {
  const bodyMatch = /enum\s+\w+\s*\{([\s\S]*?)\}/.exec(source);
  if (!bodyMatch) return [];
  return (bodyMatch[1] ?? "")
    .split(",")
    .map((s) => s.trim())
    .filter((s) => s.length > 0 && !s.startsWith("//") && !s.startsWith("/*"))
    .map((s) => s.replace(/\(.*?\)/, "").trim())
    .filter((s) => /^[A-Za-z_][A-Za-z0-9_]*$/.test(s));
}

interface ImportInfo {
  className: string;
  packagePath: string;
}

export function extractImports(source: string): ImportInfo[] {
  const out: ImportInfo[] = [];
  const re = /^import\s+([\w.]+);\s*$/gm;
  let m: RegExpExecArray | null;
  while ((m = re.exec(source)) !== null) {
    const full = m[1] ?? "";
    const className = full.split(".").pop() ?? "";
    const packagePath = full.slice(0, full.length - className.length - 1);
    out.push({ className, packagePath });
  }
  return out;
}

function findJavaFile(root: string, className: string): string | null {
  const candidates = ["src/main/java", "src/test/java", "src/main/kotlin"];
  for (const base of candidates) {
    const found = findInDir(path.join(root, base), className);
    if (found) return found;
  }
  return null;
}

// Package-aware lookup: resolve the import's package path to a source directory
// so two classes sharing a name in different packages are disambiguated.
function findJavaFileByImport(root: string, imp: ImportInfo): string | null {
  const candidates = ["src/main/java", "src/test/java", "src/main/kotlin"];
  const pkgSegments = imp.packagePath ? imp.packagePath.split(".") : [];
  if (pkgSegments.length > 0) {
    for (const base of candidates) {
      const abs = path.join(root, base, ...pkgSegments, `${imp.className}.java`);
      if (fs.existsSync(abs)) return abs;
    }
  }
  return findJavaFile(root, imp.className);
}

function findInDir(dir: string, className: string): string | null {
  if (!fs.existsSync(dir)) return null;
  const direct = path.join(dir, `${className}.java`);
  if (fs.existsSync(direct)) return direct;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.isDirectory()) {
      const found = findInDir(path.join(dir, entry.name), className);
      if (found) return found;
    }
  }
  return null;
}

export const extractApiSpecTool = {
  name: "extract_api_spec",
  description: "Extract API specification from Spring MVC controller. Deterministic — never fabricates.",
  parameters: SCHEMA,

  async execute(params: unknown, ctx: ToolContext): Promise<ToolResult<ApiSpecOutput>> {
    const started = Date.now();
    try {
      const issues = validateSchema(SCHEMA, params);
      if (issues.length > 0) {
        throw invalidInput(`invalid input: ${issues.map((i) => `${i.path} ${i.message}`).join("; ")}`);
      }
      const input = params as ExtractApiSpecInput;
      const abs = resolveInRoot(ctx.projectRoot, input.controllerFile);
      const source = fs.readFileSync(abs, "utf8");

      const { controllerName, basePath } = parseClassInfo(source);
      const endpoints = parseEndpoints(source).map((ep) => ({ ...ep, path: joinPaths(basePath, ep.path) }));
      const imports = extractImports(source);

      const dtos: Record<string, { fields: FieldInfo[] }> = {};
      const enums: Record<string, { values: string[] }> = {};

      for (const imp of imports) {
        const file = findJavaFileByImport(ctx.projectRoot, imp);
        if (!file) continue;
        const rel = path.relative(ctx.projectRoot, file).replace(/\\/g, "/");
        const src = fs.readFileSync(file, "utf8");
        if (/enum\s+\w+/.test(src)) {
          enums[imp.className] = { values: parseEnumValues(src) };
        } else {
          dtos[imp.className] = { fields: parseFieldDeclarations(src) };
        }
      }

      // Attach DTO fields to request bodies.
      for (const ep of endpoints) {
        if (ep.requestBody) {
          const dto = dtos[ep.requestBody.type];
          if (dto) ep.requestBody.fields = dto.fields;
        }
      }

      const output: ApiSpecOutput = { controllerName, basePath, endpoints, dtos, enums };
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, targetPath: input.controllerFile, durationMs: Date.now() - started, ok: true });
      return ok(output);
    } catch (e) {
      const toolErr = toToolError(e, "extract_api_spec failed");
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, targetPath: (params as ExtractApiSpecInput)?.controllerFile, durationMs: Date.now() - started, ok: false, errorCategory: toolErr.category });
      return err(toolErr);
    }
  },
};
