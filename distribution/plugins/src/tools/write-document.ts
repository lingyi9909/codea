import { writeFileAtomic, type WriteResult } from "./filesystem";
import { invalidInput } from "./errors";
import { toToolError } from "./failure-classifier";
import { validateSchema, type JsonSchema } from "./schemas";
import { err, ok, type ToolContext, type ToolResult } from "./types";

// API Doc write tool. Path restricted to the fixed docs roots docs/, doc/,
// api-docs/ — the caller cannot nominate a docs root. DLP-gated and atomic.
// Never writes to src/, .git/ or outside the project root.

export interface WriteDocumentInput {
  path: string;
  content: string;
}

const DEFAULT_DOCS_ROOTS = ["docs", "doc", "api-docs"];

const SCHEMA: JsonSchema = {
  type: "object",
  properties: {
    path: { type: "string", minLength: 1 },
    content: { type: "string" },
  },
  required: ["path", "content"],
  additionalProperties: false,
};

export const writeDocumentTool = {
  name: "write_document",
  description: "Write documentation file. Path restricted to docs/, doc/, api-docs/.",
  parameters: SCHEMA,

  async execute(params: unknown, ctx: ToolContext): Promise<ToolResult<WriteResult>> {
    const started = Date.now();
    try {
      const issues = validateSchema(SCHEMA, params);
      if (issues.length > 0) {
        throw invalidInput(`invalid input: ${issues.map((i) => `${i.path} ${i.message}`).join("; ")}`);
      }
      const input = params as WriteDocumentInput;

      const result = writeFileAtomic({
        projectRoot: ctx.projectRoot,
        relPath: input.path,
        content: input.content,
        allowedRoots: DEFAULT_DOCS_ROOTS,
        overwrite: true,
      });

      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "write", projectRoot: ctx.projectRoot, targetPath: input.path, durationMs: Date.now() - started, ok: true });
      return ok(result);
    } catch (e) {
      const toolErr = toToolError(e, "write_document failed");
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "write", projectRoot: ctx.projectRoot, targetPath: (params as WriteDocumentInput)?.path, durationMs: Date.now() - started, ok: false, errorCategory: toolErr.category });
      return err(toolErr);
    }
  },
};
