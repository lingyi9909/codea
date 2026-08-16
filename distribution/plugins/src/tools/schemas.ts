// Lightweight JSON Schema validation for tool inputs. Deliberately small — only
// the constructs the 7 tools use (object/string/number/integer/boolean/array,
// enum, required, min/max, length, nested properties, additionalProperties).

export interface SchemaProperty {
  type?: string | string[];
  enum?: unknown[];
  minimum?: number;
  maximum?: number;
  minLength?: number;
  maxLength?: number;
  items?: SchemaProperty;
  properties?: Record<string, SchemaProperty>;
  required?: string[];
  additionalProperties?: boolean;
}

export interface JsonSchema {
  type: string;
  properties?: Record<string, SchemaProperty>;
  required?: string[];
  additionalProperties?: boolean;
}

export interface ValidationIssue {
  path: string;
  message: string;
}

function typeMatches(value: unknown, type: string): boolean {
  switch (type) {
    case "string":
      return typeof value === "string";
    case "number":
      return typeof value === "number";
    case "integer":
      return typeof value === "number" && Number.isInteger(value);
    case "boolean":
      return typeof value === "boolean";
    case "array":
      return Array.isArray(value);
    case "object":
      return typeof value === "object" && value !== null && !Array.isArray(value);
    case "null":
      return value === null;
    default:
      return true;
  }
}

function checkProperty(value: unknown, prop: SchemaProperty, path: string, issues: ValidationIssue[]): void {
  if (value === undefined) return;

  if (prop.type !== undefined) {
    const types = Array.isArray(prop.type) ? prop.type : [prop.type];
    if (!types.some((t) => typeMatches(value, t))) {
      issues.push({ path, message: `expected ${types.join("|")}, got ${typeof value}` });
      return;
    }
  }

  if (prop.enum !== undefined && !prop.enum.some((e) => e === value)) {
    issues.push({ path, message: `must be one of ${JSON.stringify(prop.enum)}` });
  }

  if (typeof value === "number") {
    if (prop.minimum !== undefined && value < prop.minimum) {
      issues.push({ path, message: `must be >= ${prop.minimum}` });
    }
    if (prop.maximum !== undefined && value > prop.maximum) {
      issues.push({ path, message: `must be <= ${prop.maximum}` });
    }
  }

  if (typeof value === "string") {
    if (prop.minLength !== undefined && value.length < prop.minLength) {
      issues.push({ path, message: `length must be >= ${prop.minLength}` });
    }
    if (prop.maxLength !== undefined && value.length > prop.maxLength) {
      issues.push({ path, message: `length must be <= ${prop.maxLength}` });
    }
  }

  if (Array.isArray(value) && prop.items) {
    value.forEach((item, i) => checkProperty(item, prop.items as SchemaProperty, `${path}[${i}]`, issues));
  }

  if (value !== null && typeof value === "object" && !Array.isArray(value) && prop.properties) {
    const obj = value as Record<string, unknown>;
    for (const key of prop.required ?? []) {
      if (obj[key] === undefined) {
        issues.push({ path: `${path}.${key}`, message: "required" });
      }
    }
    for (const [key, child] of Object.entries(obj)) {
      if (child === undefined) continue;
      if (prop.additionalProperties === false && !(key in prop.properties)) {
        issues.push({ path: `${path}.${key}`, message: "not allowed" });
        continue;
      }
      const childSchema = prop.properties[key];
      if (childSchema) checkProperty(child, childSchema, `${path}.${key}`, issues);
    }
  }
}

export function validateSchema(schema: JsonSchema, value: unknown): ValidationIssue[] {
  const issues: ValidationIssue[] = [];

  if (schema.type === "object") {
    if (value === null || typeof value !== "object" || Array.isArray(value)) {
      return [{ path: "$", message: "expected object" }];
    }
    const obj = value as Record<string, unknown>;
    for (const key of schema.required ?? []) {
      if (obj[key] === undefined) {
        issues.push({ path: `$.${key}`, message: "required" });
      }
    }
    if (schema.additionalProperties === false) {
      for (const key of Object.keys(obj)) {
        if (!(key in (schema.properties ?? {}))) {
          issues.push({ path: `$.${key}`, message: "not allowed" });
        }
      }
    }
    for (const [key, child] of Object.entries(obj)) {
      if (child === undefined) continue;
      const childSchema = schema.properties?.[key];
      if (childSchema) checkProperty(child, childSchema, `$.${key}`, issues);
    }
  } else {
    checkProperty(value, { type: schema.type }, "$", issues);
  }

  return issues;
}

export function isValid(schema: JsonSchema, value: unknown): boolean {
  return validateSchema(schema, value).length === 0;
}
