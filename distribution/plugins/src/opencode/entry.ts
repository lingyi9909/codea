import * as os from "node:os";
import * as path from "node:path";
import { z } from "zod";
import { AuditLogger } from "../audit-log";
import { RuntimeSecurityGuard } from "../runtime-security-guard";
import { validateNativeReadPath } from "../security/path-policy";
import { DifyClient, difyConfigFromEnv } from "../dify-query";
import { collectReviewContextTool } from "../tools/collect-review-context";
import { analyzeTestProjectTool } from "../tools/analyze-test-project";
import { writeTestFileTool } from "../tools/write-test-file";
import { runProjectTestTool } from "../tools/run-project-test";
import { extractApiSpecTool } from "../tools/extract-api-spec";
import { validateApiExampleTool } from "../tools/validate-api-example";
import { writeDocumentTool } from "../tools/write-document";
import type { ToolContext as CodeaToolContext, ToolResult as CodeaToolResult, WriteOwnership } from "../tools/types";
import type { Hooks, PluginModule, ToolContext, ToolDefinition, ToolResult } from "./types";

// OpenCode v1.18.11 plugin adapter. This is the real integration point: it
// default-exports a plugin module that OpenCode loads, registers the 7 enterprise
// custom tools plus dify-query, and wires RuntimeSecurityGuard into the before
// path (deny aborts, write/execute enters permission) and the output path
// (DLP redact/block before the result reaches the model).

const TOOL_ACTIONS: Record<string, string> = {
  collect_review_context: "read",
  analyze_test_project: "read",
  write_test_file: "write",
  run_project_test: "execute",
  extract_api_spec: "read",
  validate_api_example: "read",
  write_document: "write",
  "dify-query": "read",
};

const APPROVAL_ACTIONS = new Set(["write", "execute"]);

// Native OpenCode tools whose output must pass output DLP before reaching the
// model. Enterprise agents are allowed read/grep/glob/bash, so their raw output
// (file contents, grep matches, command output) must be scanned for secrets.
const NATIVE_OUTPUT_DLP_TOOLS = new Set(["read", "grep", "glob", "bash"]);

// Native tools that carry a file/directory path in their args. Read-style tools
// and mutation tools share the same containment/sensitive-target validator.
const NATIVE_PATH_TOOLS = new Set(["read", "grep", "glob", "write", "edit"]);
const NATIVE_MUTATION_TOOLS = new Set(["write", "edit"]);

function nativePathFor(tool: string, args: any): string | undefined {
  if (tool === "read" || tool === "write" || tool === "edit") return args?.filePath;
  if (tool === "grep" || tool === "glob") return args?.path;
  return undefined;
}

const TOOL_ARGS: Record<string, z.ZodRawShape> = {
  collect_review_context: {
    source: z.enum(["staged", "unstaged", "base-branch", "commit", "range", "file-path"]),
    baseBranch: z.string().optional(),
    commit: z.string().optional(),
    rangeFrom: z.string().optional(),
    rangeTo: z.string().optional(),
    filePath: z.string().optional(),
  },
  analyze_test_project: {},
  write_test_file: {
    path: z.string().min(1),
    content: z.string(),
    overwrite: z.boolean().optional(),
  },
  run_project_test: {
    buildSystem: z.enum(["maven", "gradle"]),
    module: z.string().optional(),
    testClass: z.string().optional(),
    testMethod: z.string().optional(),
    profiles: z.array(z.string()).optional(),
    timeoutSeconds: z.number().int().min(1).optional(),
  },
  extract_api_spec: {
    controllerFile: z.string().min(1),
  },
  validate_api_example: {
    example: z.record(z.string(), z.unknown()),
    spec: z.record(z.string(), z.unknown()),
    endpointIndex: z.number().int().min(0),
  },
  write_document: {
    path: z.string().min(1),
    content: z.string(),
  },
  "dify-query": {
    question: z.string().min(1),
  },
};

function targetPathFor(tool: string, args: any): string | undefined {
  if (tool === "write_test_file" || tool === "write_document") return args?.path;
  if (tool === "extract_api_spec") return args?.controllerFile;
  if (tool === "collect_review_context") return args?.filePath;
  return undefined;
}

type OwnershipFactory = (sessionId: string, agent: string) => WriteOwnership;

function toCodeaContext(
  octx: ToolContext,
  audit: AuditLogger,
  guard: RuntimeSecurityGuard,
  ownership?: WriteOwnership,
): CodeaToolContext {
  return {
    sessionId: octx.sessionID,
    agent: octx.agent,
    projectRoot: octx.directory,
    audit,
    guard,
    ownership,
  };
}

type CodeaTool = {
  name: string;
  description?: string;
  execute(params: unknown, ctx: CodeaToolContext): Promise<CodeaToolResult<unknown>>;
};

function adaptTool(
  name: string,
  codeaTool: CodeaTool,
  audit: AuditLogger,
  guard: RuntimeSecurityGuard,
  ownershipFactory?: OwnershipFactory,
): ToolDefinition {
  return {
    description: codeaTool.description ?? name,
    args: TOOL_ARGS[name] ?? {},
    async execute(args: any, octx: ToolContext): Promise<ToolResult> {
      const action = TOOL_ACTIONS[name] ?? "read";
      const ownership = ownershipFactory ? ownershipFactory(octx.sessionID, octx.agent) : undefined;
      const codeaCtx = toCodeaContext(octx, audit, guard, ownership);

      // 1. guard before: path policy + DLP input. Deny aborts the call.
      const before = guard.before({
        sessionId: octx.sessionID,
        agent: octx.agent,
        tool: name,
        action,
        projectRoot: octx.directory,
        targetPath: targetPathFor(name, args),
        input: args,
      });
      if (before.decision === "deny") {
        throw new Error(before.reason ?? `${name} denied by security policy`);
      }

      // 2. write/execute requires runtime approval (enters the permission flow).
      if (APPROVAL_ACTIONS.has(action)) {
        await octx.ask({
          permission: name,
          patterns: ["*"],
          always: ["*"],
          metadata: { tool: name, action },
        });
      }

      // 3. run the enterprise tool.
      const result = await codeaTool.execute(args, codeaCtx);

      // 4. output DLP: redact/block before returning to the model.
      const raw = result.ok
        ? JSON.stringify(result.data)
        : JSON.stringify({ error: { category: result.error.category, message: result.error.message } });
      const dlp = guard.guardOutput(raw);

      return {
        title: name,
        output: dlp.output,
        metadata: {
          ok: result.ok,
          ...(result.ok ? {} : { errorCategory: result.error.category }),
          dlpBlocked: dlp.blocked,
          ...(dlp.rule ? { dlpRule: dlp.rule } : {}),
        },
      };
    },
  };
}

function buildDifyTool(dify: DifyClient | null, audit: AuditLogger, guard: RuntimeSecurityGuard): ToolDefinition {
  return {
    description: "Query the intranet Dify knowledge base. Degrades gracefully when unavailable.",
    args: TOOL_ARGS["dify-query"] ?? {},
    async execute(args: any, octx: ToolContext): Promise<ToolResult> {
      const action = TOOL_ACTIONS["dify-query"] ?? "read";
      const codeaCtx = toCodeaContext(octx, audit, guard);
      const before = guard.before({
        sessionId: octx.sessionID,
        agent: octx.agent,
        tool: "dify-query",
        action,
        projectRoot: octx.directory,
        input: args,
      });
      if (before.decision === "deny") {
        throw new Error(before.reason ?? "dify-query denied by security policy");
      }

      const question = typeof args?.question === "string" ? args.question : "";
      const result = dify
        ? await dify.query(question)
        : { degraded: true, error: "dify-not-configured" };
      guard.after({
        sessionId: octx.sessionID,
        agent: octx.agent,
        tool: "dify-query",
        action,
        projectRoot: octx.directory,
        durationMs: 0,
        ok: !result.degraded,
      });

      const dlp = guard.guardOutput(JSON.stringify(result));
      return {
        title: "dify-query",
        output: dlp.output,
        metadata: { degraded: result.degraded, dlpBlocked: dlp.blocked },
      };
    },
  };
}

export const plugin: PluginModule = {
  id: "codea-enterprise",
  server: async (input, options) => {
    const opts = options ?? {};
    const auditLog =
      (typeof opts.auditLog === "string" && opts.auditLog) ||
      process.env.CODEA_AUDIT_LOG ||
      path.join(os.tmpdir(), "codea-audit.log");
    const audit = new AuditLogger(auditLog, input.directory);
    const guard = new RuntimeSecurityGuard(audit);

    const difyEnv = difyConfigFromEnv(process.env);
    const dify = difyEnv
      ? new DifyClient({ baseUrl: difyEnv.baseUrl, apiKey: difyEnv.apiKey })
      : null;

    // Server-side write ownership: files created by one (session, agent) run.
    // write_test_file may only overwrite a path this exact run created, so
    // "never overwrite an existing test" holds even if the model lies about
    // overwrite=true.
    const ownershipStore = new Map<string, Set<string>>();
    const ownershipFactory: OwnershipFactory = (sessionId, agent) => {
      const key = `${sessionId}\u0000${agent}`;
      let set = ownershipStore.get(key);
      if (!set) {
        set = new Set<string>();
        ownershipStore.set(key, set);
      }
      return {
        record: (absPath) => {
          set.add(absPath);
        },
        owns: (absPath) => set.has(absPath),
      };
    };

    const tools: Record<string, ToolDefinition> = {
      collect_review_context: adaptTool("collect_review_context", collectReviewContextTool, audit, guard),
      analyze_test_project: adaptTool("analyze_test_project", analyzeTestProjectTool, audit, guard),
      write_test_file: adaptTool("write_test_file", writeTestFileTool, audit, guard, ownershipFactory),
      run_project_test: adaptTool("run_project_test", runProjectTestTool, audit, guard),
      extract_api_spec: adaptTool("extract_api_spec", extractApiSpecTool, audit, guard),
      validate_api_example: adaptTool("validate_api_example", validateApiExampleTool, audit, guard),
      write_document: adaptTool("write_document", writeDocumentTool, audit, guard),
      "dify-query": buildDifyTool(dify, audit, guard),
    };

    const hooks: Hooks = {
      tool: tools,
      "tool.execute.before": async (hookInput, output) => {
        const tool = hookInput.tool;
        // Intercept the native bash tool: deny dangerous commands. "ask" decisions
        // are handled natively by the bash tool's own permission flow.
        if (tool === "bash") {
          const command = (output.args as any)?.command;
          if (typeof command !== "string") return;
          const decision = guard.before({
            sessionId: hookInput.sessionID,
            agent: "",
            tool: "bash",
            action: "execute",
            projectRoot: input.directory,
            command,
          });
          if (decision.decision === "deny") {
            throw new Error(decision.reason ?? "command denied");
          }
          return;
        }

        // Every native path-bearing tool stays inside the project and may not
        // target credential/sensitive paths. OpenCode commonly sends absolute
        // filePath values, so use the native validator before the generic guard.
        if (NATIVE_PATH_TOOLS.has(tool)) {
          const targetPath = nativePathFor(tool, output.args);
          if (typeof targetPath === "string" && targetPath !== "") {
            const reason = validateNativeReadPath(input.directory, targetPath);
            if (reason) {
              throw new Error(`native-path:${reason}`);
            }
          }
        }

        // Debug can request native write/edit through OpenCode's normal `ask`
        // permission flow. Before that approval/execution occurs, Codea still
        // scans the complete mutation payload for DLP violations. The path was
        // validated above; targetPath is intentionally omitted here so absolute
        // in-root OpenCode paths work consistently on Windows and POSIX.
        if (NATIVE_MUTATION_TOOLS.has(tool)) {
          const decision = guard.before({
            sessionId: hookInput.sessionID,
            agent: "",
            tool,
            action: "write",
            projectRoot: input.directory,
            input: output.args,
          });
          if (decision.decision === "deny") {
            throw new Error(decision.reason ?? `${tool} denied by security policy`);
          }
        }
      },
      "tool.execute.after": async (hookInput, output) => {
        // Native tool output (file contents / grep matches / command output) must
        // pass output DLP before it reaches the model. Layer-1 secrets block the
        // whole output; ordinary sensitive values are redacted in place.
        if (!NATIVE_OUTPUT_DLP_TOOLS.has(hookInput.tool)) return;
        const dlp = guard.guardOutput(output.output);
        output.output = dlp.output;
        if (dlp.blocked) {
          output.metadata = { ...(output.metadata ?? {}), dlpBlocked: true, dlpRule: dlp.rule };
        }
      },
    };

    return hooks;
  },
};

export default plugin;
