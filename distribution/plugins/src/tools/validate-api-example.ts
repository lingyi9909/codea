import { invalidInput } from "./errors";
import { toToolError } from "./failure-classifier";
import { validateSchema, type JsonSchema } from "./schemas";
import type { ApiSpecOutput, FieldInfo } from "./extract-api-spec";
import { err, ok, type ToolContext, type ToolResult } from "./types";

// API Doc validation tool: checks that a generated request/response example
// matches the spec produced by extract_api_spec. Pure validation — it never
// mutates the example.

export interface ValidateExampleInput {
  example: unknown;
  spec: ApiSpecOutput;
  endpointIndex: number;
}

export interface ValidationResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
}

const SCHEMA: JsonSchema = {
  type: "object",
  properties: {
    example: {},
    spec: { type: "object" },
    endpointIndex: { type: "integer", minimum: 0 },
  },
  required: ["example", "spec", "endpointIndex"],
};

function fieldTypeToJson(field: FieldInfo): string {
  const t = field.type.toLowerCase();
  if (/long|integer|int\b/.test(t)) return "number";
  if (/double|float|decimal|bigdecimal/.test(t)) return "number";
  if (/boolean/.test(t)) return "boolean";
  if (/list|set|array|collection|<\s*>/.test(t) || /<.+>/.test(t)) return "array";
  return "string";
}

function isEnumType(spec: ApiSpecOutput, type: string): boolean {
  return Object.prototype.hasOwnProperty.call(spec.enums, type);
}

export function validateExample(input: ValidateExampleInput): ValidationResult {
  const { example, spec, endpointIndex } = input;
  const errors: string[] = [];
  const warnings: string[] = [];

  const endpoint = spec.endpoints[endpointIndex];
  if (!endpoint) {
    return { valid: false, errors: [`endpointIndex ${endpointIndex} out of range`], warnings };
  }

  if (!endpoint.requestBody) {
    return { valid: true, errors, warnings: ["endpoint has no request body to validate against"] };
  }

  const dto = spec.dtos[endpoint.requestBody.type];
  if (!dto) {
    return { valid: false, errors: [`request body type ${endpoint.requestBody.type} has no extracted DTO`], warnings };
  }

  const exampleObj = example as Record<string, unknown>;
  if (typeof exampleObj !== "object" || exampleObj === null || Array.isArray(exampleObj)) {
    return { valid: false, errors: ["example must be an object"], warnings };
  }

  const fieldByName = new Map(dto.fields.map((f) => [f.name, f]));

  // unknown fields: a field not present in the extracted DTO is a fabrication
  // against the spec, so it is an error, not a warning.
  for (const key of Object.keys(exampleObj)) {
    if (!fieldByName.has(key)) {
      errors.push(`unknown field "${key}" (not in extracted DTO)`);
    }
  }

  // required + validation
  for (const field of dto.fields) {
    const value = exampleObj[field.name];
    const required = field.validation.some((v) => v.startsWith("@NotNull") || v.startsWith("@NotBlank") || v.startsWith("@NotEmpty"));

    if (required && (value === undefined || value === null || value === "")) {
      errors.push(`missing required field "${field.name}"`);
      continue;
    }
    if (value === undefined) continue;

    if (isEnumType(spec, field.type)) {
      const allowed = spec.enums[field.type]?.values ?? [];
      if (!allowed.includes(String(value))) {
        errors.push(`field "${field.name}" value "${String(value)}" not in enum ${field.type}`);
      }
      continue;
    }

    const jsonType = fieldTypeToJson(field);
    const actualType = Array.isArray(value) ? "array" : typeof value;
    if (jsonType === "number" && actualType !== "number") {
      errors.push(`field "${field.name}" expected number, got ${actualType}`);
    }
    if (jsonType === "boolean" && actualType !== "boolean") {
      errors.push(`field "${field.name}" expected boolean, got ${actualType}`);
    }

    if (typeof value === "number") {
      const min = /@Min\((\d+)\)/.exec(field.validation.join(" "));
      if (min && value < parseInt(min[1] ?? "0", 10)) {
        errors.push(`field "${field.name}" below @Min(${min[1]})`);
      }
      const max = /@Max\((\d+)\)/.exec(field.validation.join(" "));
      if (max && value > parseInt(max[1] ?? "0", 10)) {
        errors.push(`field "${field.name}" above @Max(${max[1]})`);
      }
    }
  }

  return { valid: errors.length === 0, errors, warnings };
}

export const validateApiExampleTool = {
  name: "validate_api_example",
  description: "Validate that generated API examples match the extracted spec schema.",
  parameters: SCHEMA,

  async execute(params: unknown, ctx: ToolContext): Promise<ToolResult<ValidationResult>> {
    const started = Date.now();
    try {
      const issues = validateSchema(SCHEMA, params);
      if (issues.length > 0) {
        throw invalidInput(`invalid input: ${issues.map((i) => `${i.path} ${i.message}`).join("; ")}`);
      }
      const input = params as ValidateExampleInput;
      const result = validateExample(input);
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: true });
      return ok(result);
    } catch (e) {
      const toolErr = toToolError(e, "validate_api_example failed");
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: false, errorCategory: toolErr.category });
      return err(toolErr);
    }
  },
};
