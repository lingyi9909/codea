import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as path from "node:path";
import { validatePermissions, type PermissionsConfig } from "../src/permissions";

const PERMISSIONS_PATH = path.join(import.meta.dir, "..", "..", "config", "opencode", "permissions.json");

function loadActual(): PermissionsConfig {
  return JSON.parse(fs.readFileSync(PERMISSIONS_PATH, "utf8")) as PermissionsConfig;
}

describe("validatePermissions — real config", () => {
  test("passes the actual permissions.json", () => {
    const issues = validatePermissions(loadActual());
    expect(issues).toEqual([]);
  });
});

describe("validatePermissions — invariants", () => {
  test("enterprise agent with native bash allow is rejected", () => {
    const cfg: PermissionsConfig = {
      agents: {
        general: { write: "ask", edit: "ask", bash: "ask" },
        "code-reviewer": { read: "allow", bash: "allow", write: "deny", edit: "deny" },
        "unit-test-generator": { write: "deny", edit: "deny", bash: "deny" },
        "api-documentation": { write: "deny", edit: "deny", bash: "deny" },
      },
    };
    const issues = validatePermissions(cfg);
    expect(issues.some((i) => i.agent === "code-reviewer" && i.message.includes("bash"))).toBe(true);
  });

  test("enterprise agent with write not deny is rejected", () => {
    const cfg: PermissionsConfig = {
      agents: {
        general: { write: "ask", edit: "ask", bash: "ask" },
        "code-reviewer": { write: "deny", edit: "deny", bash: "deny" },
        "unit-test-generator": { write: "ask", edit: "deny", bash: "deny" },
        "api-documentation": { write: "deny", edit: "deny", bash: "deny" },
      },
    };
    const issues = validatePermissions(cfg);
    expect(issues.some((i) => i.agent === "unit-test-generator" && i.message.includes("write"))).toBe(true);
  });

  test("general with bash deny is rejected (native capability regression)", () => {
    const cfg: PermissionsConfig = {
      agents: {
        general: { write: "ask", edit: "ask", bash: "deny" },
        "code-reviewer": { write: "deny", edit: "deny", bash: "deny" },
        "unit-test-generator": { write: "deny", edit: "deny", bash: "deny" },
        "api-documentation": { write: "deny", edit: "deny", bash: "deny" },
      },
    };
    const issues = validatePermissions(cfg);
    expect(issues.some((i) => i.agent === "general")).toBe(true);
  });
});
