import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import type { ToolContext } from "../src/tools/types";
import { createVerifyProjectTool } from "../src/tools/verify-project";

function context(root: string): ToolContext {
  return {
    sessionId: "task30-local-smoke",
    rootTurnId: "root-local-smoke",
    agent: "debug",
    projectRoot: root,
    audit: {} as any,
    guard: { after() {} } as any,
  };
}

describe("Task 30 real local verification smoke", () => {
  test("executes a standard-library-only Go project with the real runner", async () => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "codea-task30-go-smoke-"));
    try {
      fs.writeFileSync(path.join(root, "go.mod"), "module example.com/task30smoke\n\ngo 1.22\n", "utf8");
      fs.writeFileSync(path.join(root, "math.go"), "package smoke\n\nfunc Add(a, b int) int { return a + b }\n", "utf8");
      fs.writeFileSync(path.join(root, "math_test.go"), "package smoke\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) { if Add(2, 3) != 5 { t.Fatal(\"bad sum\") } }\n", "utf8");

      const out = await createVerifyProjectTool().execute({ timeoutSeconds: 30 }, context(root));
      expect(out.ok).toBe(true);
      expect(out.data.result).toBe("PASS");
      expect(out.data.profile).toBe("go");
      expect(out.data.stages.map((stage: any) => [stage.name, stage.category])).toEqual([
        ["test", "PASS"],
        ["vet", "PASS"],
      ]);
      expect(out.data.stages.map((stage: any) => stage.commandSummary)).toEqual([
        "go test ./...",
        "go vet ./...",
      ]);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });
});
