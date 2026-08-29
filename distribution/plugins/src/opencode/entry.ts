import * as os from "node:os";
import * as path from "node:path";
import { z } from "zod";
import { AuditLogger } from "../audit-log";
import { RuntimeSecurityGuard } from "../runtime-security-guard";
import { validateNativeReadPath } from "../security/path-policy";
import { DifyClient, difyConfigFromEnv } from "../dify-query";
import { TaskStateStore } from "../task-state/store";
import { RootTurnEpochs } from "../task-state/epoch";
import { requirePlan } from "../task-state/gate";
import { collectReviewContextTool } from "../tools/collect-review-context";
import { analyzeTestProjectTool } from "../tools/analyze-test-project";
import { writeTestFileTool } from "../tools/write-test-file";
import { runProjectTestTool } from "../tools/run-project-test";
import { extractApiSpecTool } from "../tools/extract-api-spec";
import { validateApiExampleTool } from "../tools/validate-api-example";
import { writeDocumentTool } from "../tools/write-document";
import { createTaskPlanTool } from "../tools/task-plan";
import { createTaskStepTool } from "../tools/task-step";
import { createTaskStatusTool } from "../tools/task-status";
import { verifyProjectTool } from "../tools/verify-project";
import type { ToolContext as CodeaToolContext, ToolResult as CodeaToolResult, WriteOwnership } from "../tools/types";
import type { Hooks, PluginModule, ToolContext, ToolDefinition, ToolResult } from "./types";

const CODEA_PLUGIN_ID = "codea-enterprise";

const TOOL_ACTIONS: Record<string, string> = {
  collect_review_context: "read",
  analyze_test_project: "read",
  write_test_file: "write",
  run_project_test: "execute",
  verify_project: "execute",
  extract_api_spec: "read",
  validate_api_example: "read",
  write_document: "write",
  task_plan: "plan",
  task_step: "plan",
  task_status: "plan",
  "dify-query": "read",
};

const APPROVAL_ACTIONS = new Set(["write", "execute"]);
const NATIVE_OUTPUT_DLP_TOOLS = new Set(["read", "grep", "glob", "bash"]);
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
    path: z.string().min(1), content: z.string(), overwrite: z.boolean().optional(),
  },
  run_project_test: {
    buildSystem: z.enum(["maven", "gradle"]), module: z.string().optional(), testClass: z.string().optional(),
    testMethod: z.string().optional(), profiles: z.array(z.string()).optional(), timeoutSeconds: z.number().int().min(1).optional(),
  },
  verify_project: { timeoutSeconds: z.number().int().min(1).max(600).optional() },
  extract_api_spec: { controllerFile: z.string().min(1) },
  validate_api_example: {
    example: z.record(z.string(), z.unknown()), spec: z.record(z.string(), z.unknown()), endpointIndex: z.number().int().min(0),
  },
  write_document: { path: z.string().min(1), content: z.string() },
  task_plan: {
    goal: z.string().min(1).max(1000),
    steps: z.array(z.object({
      id: z.string().min(1).max(100),
      title: z.string().min(1).max(300),
      verification: z.string().max(500).optional(),
    }).strict()).min(3).max(7),
  },
  task_step: {
    stepId: z.string().min(1).max(100),
    status: z.enum(["in_progress", "completed", "blocked"]),
    evidence: z.string().max(1000).optional(),
  },
  task_status: {},
  "dify-query": { question: z.string().min(1) },
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
  rootTurnId: string,
  ownership?: WriteOwnership,
): CodeaToolContext {
  return { sessionId: octx.sessionID, rootTurnId, agent: octx.agent, projectRoot: octx.directory, audit, guard, ownership };
}

type CodeaTool = {
  name: string;
  description?: string;
  execute(params: unknown, ctx: CodeaToolContext): Promise<CodeaToolResult<unknown>>;
};

function planningMetadata(name: string, result: CodeaToolResult<unknown>): Record<string, string> {
  if (!result.ok || !["task_plan", "task_step", "task_status"].includes(name)) return {};
  const steps = (result.data as any)?.plan?.steps;
  if (!Array.isArray(steps)) return {};
  const completed = steps.filter((step: any) => step?.status === "completed").length;
  const active = steps.find((step: any) => step?.status === "in_progress");
  return {
    codeaTaskPlan: "true",
    codeaPlanTotal: String(steps.length),
    codeaPlanCompleted: String(completed),
    codeaPlanActive: typeof active?.id === "string" ? active.id : "",
  };
}

function verificationMetadata(name: string, result: CodeaToolResult<unknown>): Record<string, string> {
  if (name !== "verify_project" || !result.ok) return {};
  const data = result.data as any;
  const resultValue = typeof data?.result === "string" ? data.result.toLowerCase() : "";
  const profileValue = typeof data?.profile === "string" ? data.profile.toLowerCase() : "";
  const allowedResults = new Set(["pass", "fail", "timeout", "not_configured", "error"]);
  const allowedProfiles = new Set(["maven", "gradle", "go", "unknown"]);
  if (!allowedResults.has(resultValue) || !allowedProfiles.has(profileValue)) return {};
  return {
    codeaVerification: "true",
    codeaVerificationResult: resultValue,
    codeaVerificationProfile: profileValue,
  };
}

async function requirePlanForOperation(
  taskState: TaskStateStore,
  guard: RuntimeSecurityGuard,
  sessionId: string,
  rootTurnId: string,
  agent: string,
  tool: string,
  action: string,
  projectRoot: string,
): Promise<void> {
  try {
    await requirePlan(taskState, sessionId, rootTurnId, `${tool}:${action}`);
  } catch (error: any) {
    guard.after({
      sessionId,
      agent,
      tool,
      action,
      projectRoot,
      durationMs: 0,
      ok: false,
      errorCategory: error?.category ?? "PLAN_REQUIRED",
    });
    throw error;
  }
}

function adaptTool(
  name: string,
  codeaTool: CodeaTool,
  audit: AuditLogger,
  guard: RuntimeSecurityGuard,
  taskState: TaskStateStore,
  rootTurns: RootTurnEpochs,
  ownershipFactory?: OwnershipFactory,
): ToolDefinition {
  return {
    description: codeaTool.description ?? name,
    args: TOOL_ARGS[name] ?? {},
    async execute(args: any, octx: ToolContext): Promise<ToolResult> {
      const action = TOOL_ACTIONS[name] ?? "read";
      const ownership = ownershipFactory ? ownershipFactory(octx.sessionID, octx.agent) : undefined;
      const rootTurnId = rootTurns.current(octx.sessionID);
      const codeaCtx = toCodeaContext(octx, audit, guard, rootTurnId, ownership);
      octx.metadata({ metadata: { codeaPlugin: CODEA_PLUGIN_ID } });

      if (APPROVAL_ACTIONS.has(action)) {
        await requirePlanForOperation(taskState, guard, octx.sessionID, rootTurnId, octx.agent, name, action, octx.directory);
      }

      const before = guard.before({
        sessionId: octx.sessionID, agent: octx.agent, tool: name, action,
        projectRoot: octx.directory, targetPath: targetPathFor(name, args), input: args,
      });
      if (before.decision === "deny") throw new Error(before.reason ?? `${name} denied by security policy`);

      if (APPROVAL_ACTIONS.has(action)) {
        await octx.ask({ permission: name, patterns: ["*"], always: ["*"], metadata: { tool: name, action } });
      }

      const result = await codeaTool.execute(args, codeaCtx);
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
          codeaPlugin: CODEA_PLUGIN_ID,
          ...planningMetadata(name, result),
          ...verificationMetadata(name, result),
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
      const before = guard.before({ sessionId: octx.sessionID, agent: octx.agent, tool: "dify-query", action, projectRoot: octx.directory, input: args });
      if (before.decision === "deny") throw new Error(before.reason ?? "dify-query denied by security policy");
      octx.metadata({ metadata: { codeaPlugin: CODEA_PLUGIN_ID } });
      const question = typeof args?.question === "string" ? args.question : "";
      const result = dify ? await dify.query(question) : { degraded: true, error: "dify-not-configured" };
      guard.after({ sessionId: octx.sessionID, agent: octx.agent, tool: "dify-query", action, projectRoot: octx.directory, durationMs: 0, ok: !result.degraded });
      const dlp = guard.guardOutput(JSON.stringify(result));
      return { title: "dify-query", output: dlp.output, metadata: { degraded: result.degraded, dlpBlocked: dlp.blocked, codeaPlugin: CODEA_PLUGIN_ID } };
    },
  };
}

export const plugin: PluginModule = {
  id: CODEA_PLUGIN_ID,
  server: async (input, options) => {
    const opts = options ?? {};
    const auditLog = (typeof opts.auditLog === "string" && opts.auditLog) || process.env.CODEA_AUDIT_LOG || path.join(os.tmpdir(), "codea-audit.log");
    const audit = new AuditLogger(auditLog, input.directory);
    const guard = new RuntimeSecurityGuard(audit);
    const taskState = new TaskStateStore({
      workspaceRoot: input.directory,
      codeaHome: process.env.CODEA_HOME || path.join(os.homedir(), ".codea"),
    });
    const rootTurns = new RootTurnEpochs();

    const difyEnv = difyConfigFromEnv(process.env);
    const dify = difyEnv ? new DifyClient({ baseUrl: difyEnv.baseUrl, apiKey: difyEnv.apiKey }) : null;

    const ownershipStore = new Map<string, Set<string>>();
    const ownershipFactory: OwnershipFactory = (sessionId, agent) => {
      const key = `${sessionId}\u0000${agent}`;
      let set = ownershipStore.get(key);
      if (!set) { set = new Set<string>(); ownershipStore.set(key, set); }
      return { record: (absPath) => { set!.add(absPath); }, owns: (absPath) => set!.has(absPath) };
    };

    const tools: Record<string, ToolDefinition> = {
      collect_review_context: adaptTool("collect_review_context", collectReviewContextTool, audit, guard, taskState, rootTurns),
      analyze_test_project: adaptTool("analyze_test_project", analyzeTestProjectTool, audit, guard, taskState, rootTurns),
      write_test_file: adaptTool("write_test_file", writeTestFileTool, audit, guard, taskState, rootTurns, ownershipFactory),
      run_project_test: adaptTool("run_project_test", runProjectTestTool, audit, guard, taskState, rootTurns),
      verify_project: adaptTool("verify_project", verifyProjectTool, audit, guard, taskState, rootTurns),
      extract_api_spec: adaptTool("extract_api_spec", extractApiSpecTool, audit, guard, taskState, rootTurns),
      validate_api_example: adaptTool("validate_api_example", validateApiExampleTool, audit, guard, taskState, rootTurns),
      write_document: adaptTool("write_document", writeDocumentTool, audit, guard, taskState, rootTurns),
      task_plan: adaptTool("task_plan", createTaskPlanTool(taskState), audit, guard, taskState, rootTurns),
      task_step: adaptTool("task_step", createTaskStepTool(taskState), audit, guard, taskState, rootTurns),
      task_status: adaptTool("task_status", createTaskStatusTool(taskState), audit, guard, taskState, rootTurns),
      "dify-query": buildDifyTool(dify, audit, guard),
    };

    const hooks: Hooks = {
      tool: tools,
      "chat.message": async (hookInput, output) => {
        rootTurns.observe(hookInput, output);
      },
      "tool.execute.before": async (hookInput, output) => {
        const tool = hookInput.tool;
        const currentRoot = rootTurns.current(hookInput.sessionID);
        if (tool === "bash") {
          await requirePlanForOperation(taskState, guard, hookInput.sessionID, currentRoot, "", tool, "execute", input.directory);
          const command = (output.args as any)?.command;
          if (typeof command !== "string") return;
          const decision = guard.before({ sessionId: hookInput.sessionID, agent: "", tool: "bash", action: "execute", projectRoot: input.directory, command });
          if (decision.decision === "deny") throw new Error(decision.reason ?? "command denied");
          return;
        }

        if (NATIVE_MUTATION_TOOLS.has(tool)) {
          await requirePlanForOperation(taskState, guard, hookInput.sessionID, currentRoot, "", tool, "write", input.directory);
        }

        if (NATIVE_PATH_TOOLS.has(tool)) {
          const targetPath = nativePathFor(tool, output.args);
          if (typeof targetPath === "string" && targetPath !== "") {
            const reason = validateNativeReadPath(input.directory, targetPath);
            if (reason) throw new Error(`native-path:${reason}`);
          }
        }

        if (NATIVE_MUTATION_TOOLS.has(tool)) {
          const decision = guard.before({ sessionId: hookInput.sessionID, agent: "", tool, action: "write", projectRoot: input.directory, input: output.args });
          if (decision.decision === "deny") throw new Error(decision.reason ?? `${tool} denied by security policy`);
        }
      },
      "tool.execute.after": async (hookInput, output) => {
        if (Object.prototype.hasOwnProperty.call(TOOL_ACTIONS, hookInput.tool)) {
          output.metadata = { ...(output.metadata ?? {}), codeaPlugin: CODEA_PLUGIN_ID, codeaPluginInvocationID: hookInput.callID };
        }
        if (!NATIVE_OUTPUT_DLP_TOOLS.has(hookInput.tool)) return;
        const dlp = guard.guardOutput(output.output);
        output.output = dlp.output;
        if (dlp.blocked) output.metadata = { ...(output.metadata ?? {}), dlpBlocked: true, dlpRule: dlp.rule };
      },
    };

    return hooks;
  },
};

export default plugin;
