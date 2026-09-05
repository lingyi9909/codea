import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as path from "node:path";

const AGENTS_ROOT = path.resolve(import.meta.dir, "../../agents");
const PLANNING_TOOLS = ["task_plan", "task_step", "task_status"] as const;
const PLAN_GATED_TOOLS = new Set([
  "write_test_file",
  "run_project_test",
  "write_document",
  "bash",
  "write",
  "edit",
]);

function manifestTools(agent: string): Map<string, string> {
  const manifest = fs.readFileSync(path.join(AGENTS_ROOT, agent, "manifest.yaml"), "utf8");
  const tools = new Map<string, string>();
  let inTools = false;

  for (const line of manifest.split(/\r?\n/)) {
    if (line === "tools:") {
      inTools = true;
      continue;
    }
    if (!inTools) continue;
    if (line !== "" && !line.startsWith("  ")) break;

    const match = /^  ([A-Za-z0-9_-]+):\s*(allow|ask|deny)\s*$/.exec(line);
    if (match) tools.set(match[1]!, match[2]!);
  }
  return tools;
}

describe("Task 29 agent planning capability contract", () => {
  for (const agent of ["unit-test-generator", "api-documentation", "debug"]) {
    test(`${agent} exposes planning tools before plan-gated operations`, () => {
      const tools = manifestTools(agent);
      const gated = [...tools.entries()]
        .filter(([name, decision]) => PLAN_GATED_TOOLS.has(name) && decision !== "deny")
        .map(([name]) => name);

      expect(gated.length).toBeGreaterThan(0);
      for (const planningTool of PLANNING_TOOLS) {
        expect(tools.get(planningTool)).toBe("allow");
      }
    });
  }
});
