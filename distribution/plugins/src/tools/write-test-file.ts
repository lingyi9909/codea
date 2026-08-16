import { writeFileAtomic, type WriteResult } from "./filesystem";
import { invalidInput } from "./errors";
import { toToolError } from "./failure-classifier";
import { validateSchema, type JsonSchema } from "./schemas";
import { err, ok, type ToolContext, type ToolResult } from "./types";

// Unit Test write tool. Path is restricted to detected test roots (never source
// roots), overwrite is off by default, DLP runs before write, and the write is
// atomic. Traversal/absolute/symlink escape is rejected by the path policy.

export interface WriteTestFileInput {
  path: string;
  content: string;
  overwrite?: boolean;
  testRoots?: string[];
}

const DEFAULT_TEST_ROOTS = ["src/test/java"];

const SCHEMA: JsonSchema = {
  type: "object",
  properties: {
    path: { type: "string", minLength: 1 },
    content: { type: "string" },
    overwrite: { type: "boolean" },
    testRoots: { type: "array", items: { type: "string" } },
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

      const testRoots = input.testRoots && input.testRoots.length > 0 ? input.testRoots : DEFAULT_TEST_ROOTS;

      const result = writeFileAtomic({
        projectRoot: ctx.projectRoot,
        relPath: input.path,
        content: input.content,
        allowedRoots: testRoots,
        overwrite: input.overwrite === true,
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
