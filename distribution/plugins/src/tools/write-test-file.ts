import { writeFileAtomic, type WriteResult } from "./filesystem";
import { invalidInput, notSupported } from "./errors";
import { toToolError } from "./failure-classifier";
import { validateSchema, type JsonSchema } from "./schemas";
import { err, ok, type ToolContext, type ToolResult } from "./types";
import { detectTestRoots } from "./analyze-test-project";

// Unit Test write tool. The allowed roots are derived from the real project
// layout (the same detection analyze_test_project reports), never from caller
// input — a caller cannot nominate src/main or an arbitrary root. Overwrite is
// off by default, DLP runs before write, and the write is atomic.

export interface WriteTestFileInput {
  path: string;
  content: string;
  overwrite?: boolean;
}

const SCHEMA: JsonSchema = {
  type: "object",
  properties: {
    path: { type: "string", minLength: 1 },
    content: { type: "string" },
    overwrite: { type: "boolean" },
  },
  required: ["path", "content"],
  additionalProperties: false,
};

export const writeTestFileTool = {
  name: "write_test_file",
  description: "Write a test file. Path MUST be within one of the detected test roots.",
  parameters: SCHEMA,

  async execute(params: unknown, ctx: ToolContext): Promise<ToolResult<WriteResult>> {
    const started = Date.now();
    try {
      const issues = validateSchema(SCHEMA, params);
      if (issues.length > 0) {
        throw invalidInput(`invalid input: ${issues.map((i) => `${i.path} ${i.message}`).join("; ")}`);
      }
      const input = params as WriteTestFileInput;

      const testRoots = detectTestRoots(ctx.projectRoot);
      if (testRoots.length === 0) {
        throw notSupported("no test roots detected (run analyze_test_project first)");
      }

      const result = writeFileAtomic({
        projectRoot: ctx.projectRoot,
        relPath: input.path,
        content: input.content,
        allowedRoots: testRoots,
        overwrite: input.overwrite === true,
        ownership: ctx.ownership,
      });

      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "write", projectRoot: ctx.projectRoot, targetPath: input.path, durationMs: Date.now() - started, ok: true });
      return ok(result);
    } catch (e) {
      const toolErr = toToolError(e, "write_test_file failed");
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "write", projectRoot: ctx.projectRoot, targetPath: (params as WriteTestFileInput)?.path, durationMs: Date.now() - started, ok: false, errorCategory: toolErr.category });
      return err(toolErr);
    }
  },
};
