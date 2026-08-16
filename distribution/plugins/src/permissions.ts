// Validation for distribution/config/opencode/permissions.json.
//
// Core invariant: an enterprise agent may only gain write or execute capability
// through a controlled custom tool. It must never also allow the native
// bash/write/edit tools, otherwise the custom tool's path boundary is meaningless.

export type PermissionLevel = "allow" | "ask" | "deny";
export type AgentPermissions = Record<string, PermissionLevel>;

export interface PermissionsConfig {
  agents: Record<string, AgentPermissions>;
}

export interface ValidationIssue {
  agent: string;
  message: string;
}

const ENTERPRISE_AGENTS = ["code-reviewer", "unit-test-generator", "api-documentation"] as const;
const NATIVE_WRITE_EXEC = ["write", "edit", "bash"] as const;
const READ_ONLY = new Set(["read", "grep", "glob"]);
const CONTROLLED = new Set([
  "collect_review_context",
  "analyze_test_project",
  "write_test_file",
  "run_project_test",
  "extract_api_spec",
  "validate_api_example",
  "write_document",
  "dify-query",
]);

export function validatePermissions(config: PermissionsConfig): ValidationIssue[] {
  const issues: ValidationIssue[] = [];
  if (!config || typeof config !== "object" || !config.agents) {
    return [{ agent: "", message: "missing agents map" }];
  }

  for (const name of ENTERPRISE_AGENTS) {
    const perms = config.agents[name];
    if (!perms) {
      issues.push({ agent: name, message: "missing agent permissions" });
      continue;
    }
    for (const tool of NATIVE_WRITE_EXEC) {
      if (perms[tool] !== "deny") {
        issues.push({ agent: name, message: `${tool} must be deny, got ${perms[tool] ?? "unset"}` });
      }
    }
    for (const [tool, level] of Object.entries(perms)) {
      if (level !== "allow") continue;
      if (READ_ONLY.has(tool) || CONTROLLED.has(tool)) continue;
      issues.push({ agent: name, message: `uncontrolled allow on ${tool}` });
    }
  }

  const general = config.agents["general"];
  if (!general) {
    issues.push({ agent: "general", message: "missing general permissions" });
  } else {
    for (const tool of NATIVE_WRITE_EXEC) {
      if (general[tool] === "deny") {
        issues.push({ agent: "general", message: `${tool} must not be deny (native capability)` });
      }
    }
  }

  return issues;
}
