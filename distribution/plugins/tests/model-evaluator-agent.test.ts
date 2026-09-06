import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as path from "node:path";

const AGENT_ROOT = path.resolve(import.meta.dir, "../../agents/model-evaluator");
const ALLOWED = new Set(["probe_tool_call", "probe_structured", "probe_patch", "probe_plan"]);
const MUST_DENY = new Set([
  "read", "grep", "glob", "write", "edit", "bash",
  "collect_review_context", "analyze_test_project", "write_test_file", "run_project_test",
  "verify_project", "extract_api_spec", "validate_api_example", "write_document",
  "task_plan", "task_step", "task_status", "dify-query",
]);

function manifestTools(): Map<string, string> {
  const manifest = fs.readFileSync(path.join(AGENT_ROOT, "manifest.yaml"), "utf8");
  const tools = new Map<string, string>();
  let inTools = false;
  for (const line of manifest.split(/\r?\n/)) {
    if (line === "tools:") { inTools = true; continue; }
    if (!inTools) continue;
    if (line !== "" && !line.startsWith("  ")) break;
    const match = /^  ([A-Za-z0-9_-]+):\s*(allow|ask|deny)\s*$/.exec(line);
    if (match) tools.set(match[1]!, match[2]!);
  }
  return tools;
}

describe("Task 32 model-evaluator agent contract", () => {
  test("only the four qualification probes are allowed", () => {
    const tools = manifestTools();
    for (const tool of ALLOWED) expect(tools.get(tool)).toBe("allow");
    for (const tool of MUST_DENY) expect(tools.get(tool)).toBe("deny");
    const allowed = [...tools.entries()].filter(([, decision]) => decision === "allow").map(([name]) => name).sort();
    expect(allowed).toEqual([...ALLOWED].sort());
  });

  test("fixed evaluator prompt requires ordered bounded probes and no reasoning dump", () => {
    const prompt = fs.readFileSync(path.join(AGENT_ROOT, "agent.md"), "utf8");
    const positions = ["probe_tool_call", "probe_structured", "probe_patch", "probe_plan"].map((name) => prompt.indexOf(name));
    expect(positions.every((position) => position >= 0)).toBe(true);
    expect([...positions].sort((a, b) => a - b)).toEqual(positions);
    expect(prompt).toContain("at most one retry");
    expect(prompt).toContain("Do not read project files");
    expect(prompt).toContain("Do not run commands");
    expect(prompt).toContain("Do not explain your reasoning");
  });
});
