import { resolveInRoot } from "./filesystem";
import { execCommand } from "./exec";
import { invalidInput, notSupported } from "./errors";
import { toToolError } from "./failure-classifier";
import { validateSchema, type JsonSchema } from "./schemas";
import { err, ok, type ToolContext, type ToolResult } from "./types";

// Code Reviewer tool: deterministically collects git diff context (files,
// statuses, line numbers, hunks). Runs a whitelist of read-only git commands via
// argv arrays (no shell), never leaves the project root, and bounds output size.

export type ReviewSource = "staged" | "unstaged" | "base-branch" | "commit" | "range" | "file-path";

export interface ReviewContextInput {
  source: ReviewSource;
  baseBranch?: string;
  commit?: string;
  rangeFrom?: string;
  rangeTo?: string;
  filePath?: string;
}

export interface DiffHunk {
  oldStart: number;
  oldLines: number;
  newStart: number;
  newLines: number;
  lines: string[];
}

export interface DiffFile {
  path: string;
  status: "added" | "modified" | "deleted" | "renamed";
  oldPath?: string;
  hunks: DiffHunk[];
}

export interface ReviewContextOutput {
  filesChanged: number;
  linesAdded: number;
  linesRemoved: number;
  files: DiffFile[];
}

const MAX_DIFF_BYTES = 5 * 1024 * 1024; // 5MB — explicit cap, no OOM on huge diffs

const SCHEMA: JsonSchema = {
  type: "object",
  properties: {
    source: { type: "string", enum: ["staged", "unstaged", "base-branch", "commit", "range", "file-path"] },
    baseBranch: { type: "string" },
    commit: { type: "string" },
    rangeFrom: { type: "string" },
    rangeTo: { type: "string" },
    filePath: { type: "string" },
  },
  required: ["source"],
  additionalProperties: false,
};

function buildGitDiffCommand(params: ReviewContextInput): string[] {
  switch (params.source) {
    case "staged":
      return ["git", "diff", "--cached"];
    case "unstaged":
      return ["git", "diff"];
    case "base-branch":
      return ["git", "diff", params.baseBranch ?? "origin/main"];
    case "commit":
      return ["git", "diff", `${params.commit}^`, params.commit as string];
    case "range":
      return ["git", "diff", `${params.rangeFrom}..${params.rangeTo}`];
    case "file-path":
      return ["git", "diff", "--", params.filePath as string];
  }
}

function validateInput(params: unknown): ReviewContextInput {
  const issues = validateSchema(SCHEMA, params);
  if (issues.length > 0) {
    throw invalidInput(`invalid input: ${issues.map((i) => `${i.path} ${i.message}`).join("; ")}`);
  }
  const p = params as ReviewContextInput;
  if (p.source === "commit" && !p.commit) throw invalidInput("commit is required for source=commit");
  if (p.source === "range" && (!p.rangeFrom || !p.rangeTo)) throw invalidInput("rangeFrom and rangeTo are required for source=range");
  if (p.source === "file-path" && !p.filePath) throw invalidInput("filePath is required for source=file-path");
  return p;
}

// Parses unified diff output (git diff default format) into structured files.
// Handles add/delete/rename headers and @@ hunk ranges deterministically.
export function parseUnifiedDiff(diff: string): ReviewContextOutput {
  const files: DiffFile[] = [];
  let linesAdded = 0;
  let linesRemoved = 0;

  let current: DiffFile | null = null;
  let currentHunk: DiffHunk | null = null;

  for (const raw of diff.split("\n")) {
    if (raw.startsWith("diff --git ")) {
      if (current) files.push(current);
      current = { path: "", status: "modified", hunks: [] };
      currentHunk = null;
      const m = /^diff --git a\/(.*?) b\/(.*)$/.exec(raw);
      if (m) {
        current.path = m[2] ?? "";
        current.oldPath = m[1] ?? "";
      }
      continue;
    }

    if (!current) continue;

    if (raw.startsWith("new file mode")) {
      current.status = "added";
      continue;
    }
    if (raw.startsWith("deleted file mode")) {
      current.status = "deleted";
      continue;
    }
    if (raw.startsWith("rename from ")) {
      current.status = "renamed";
      current.oldPath = raw.slice("rename from ".length).trim();
      continue;
    }
    if (raw.startsWith("rename to ")) {
      current.status = "renamed";
      current.path = raw.slice("rename to ".length).trim();
      continue;
    }

    if (raw.startsWith("@@")) {
      const m = /^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@/.exec(raw);
      if (m) {
        currentHunk = {
          oldStart: parseInt(m[1] ?? "0", 10),
          oldLines: m[2] !== undefined ? parseInt(m[2], 10) : 1,
          newStart: parseInt(m[3] ?? "0", 10),
          newLines: m[4] !== undefined ? parseInt(m[4], 10) : 1,
          lines: [],
        };
        current.hunks.push(currentHunk);
      }
      continue;
    }

    if (currentHunk) {
      currentHunk.lines.push(raw);
      if (raw.startsWith("+") && !raw.startsWith("+++")) linesAdded++;
      if (raw.startsWith("-") && !raw.startsWith("---")) linesRemoved++;
    }
  }

  if (current) files.push(current);

  return { filesChanged: files.length, linesAdded, linesRemoved, files };
}

export const collectReviewContextTool = {
  name: "collect_review_context",
  description: "Collect git diff context for code review. Returns exact file paths, line numbers, and diff hunks.",
  parameters: SCHEMA,

  async execute(params: unknown, ctx: ToolContext): Promise<ToolResult<ReviewContextOutput>> {
    const started = Date.now();
    try {
      const input = validateInput(params);

      if (input.source === "file-path") {
        // ensure the requested path is inside the project root before running git
        resolveInRoot(ctx.projectRoot, input.filePath as string);
      }

      const argv = buildGitDiffCommand(input);
      const result = await execCommand(argv, { cwd: ctx.projectRoot, timeoutMs: 30_000 });

      if (result.timedOut) {
        ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: false, errorCategory: "TIMEOUT" });
        return err(toToolError(new Error("git diff timed out"), "git diff timed out"));
      }
      if (result.exitCode !== 0) {
        ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: false, errorCategory: "COMMAND_FAILED" });
        return err(toToolError(new Error(result.stderr.trim() || `git diff failed (exit ${result.exitCode})`), "git diff failed"));
      }

      if (result.stdout.length > MAX_DIFF_BYTES) {
        ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: false, errorCategory: "NOT_SUPPORTED" });
        return err(notSupported(`diff too large (${result.stdout.length} bytes > ${MAX_DIFF_BYTES})`));
      }

      const output = parseUnifiedDiff(result.stdout);
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: true });
      return ok(output);
    } catch (e) {
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: false, errorCategory: toToolError(e, "collect_review_context failed").category });
      return err(toToolError(e, "collect_review_context failed"));
    }
  },
};
