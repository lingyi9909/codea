from pathlib import Path


def replace(path, old, new):
    p = Path(path)
    text = p.read_text()
    if old not in text:
        raise SystemExit(f"missing replacement in {path}: {old[:80]!r}")
    p.write_text(text.replace(old, new))

# Task-plan domain: bind every persisted plan to a root turn / epoch.
replace("distribution/plugins/src/task-state/types.ts",
'''export interface TaskPlan {\n  id: string;\n  sessionId: string;\n  goal: string;''',
'''export interface TaskPlan {\n  id: string;\n  sessionId: string;\n  rootMessageID: string;\n  taskEpoch: string;\n  goal: string;''')
replace("distribution/plugins/src/task-state/types.ts", "  version: 1;", "  version: 2;")

replace("distribution/plugins/src/task-state/store.ts",
'''  sessionId: z.string().min(1),\n  goal: z.string().min(1).max(TASK_PLAN_LIMITS.goalChars),''',
'''  sessionId: z.string().min(1),\n  rootMessageID: z.string().min(1),\n  taskEpoch: z.string().min(1),\n  goal: z.string().min(1).max(TASK_PLAN_LIMITS.goalChars),''')
replace("distribution/plugins/src/task-state/store.ts", "  version: z.literal(1),", "  version: z.literal(2),")
replace("distribution/plugins/src/task-state/store.ts",
'''  async create(sessionId: string, goal: string, steps: NewStep[]): Promise<TaskPlan> {\n    validateText(sessionId, "session id", 1000);\n    validateText(goal, "goal", TASK_PLAN_LIMITS.goalChars);''',
'''  async create(sessionId: string, goal: string, steps: NewStep[], rootMessageID = sessionId): Promise<TaskPlan> {\n    validateText(sessionId, "session id", 1000);\n    validateText(rootMessageID, "root message id", 1000);\n    validateText(goal, "goal", TASK_PLAN_LIMITS.goalChars);''')
replace("distribution/plugins/src/task-state/store.ts",
'''      id: randomUUID(),\n      sessionId,\n      goal,''',
'''      id: randomUUID(),\n      sessionId,\n      rootMessageID,\n      taskEpoch: rootMessageID,\n      goal,''')
replace("distribution/plugins/src/task-state/store.ts", "      version: 1,", "      version: 2,")
replace("distribution/plugins/src/task-state/store.ts",
'''    status: StepStatus,\n    evidence?: string,\n  ): Promise<TaskPlan> {\n    const plan = await this.load(sessionId);\n    if (!plan) throw new TaskStateError("TASK_STATE_INVALID", "task plan does not exist for session");''',
'''    status: StepStatus,\n    evidence?: string,\n    rootMessageID?: string,\n  ): Promise<TaskPlan> {\n    const plan = await this.load(sessionId);\n    if (!plan) throw new TaskStateError("TASK_STATE_INVALID", "task plan does not exist for session");\n    if (rootMessageID !== undefined && plan.rootMessageID !== rootMessageID) {\n      throw new TaskStateError("TASK_STATE_INVALID", "task plan root does not match current root turn");\n    }''')
replace("distribution/plugins/src/task-state/store.ts",
'''  async hasActionablePlan(sessionId: string): Promise<boolean> {\n    const plan = await this.load(sessionId);\n    if (!plan) return false;\n    return plan.steps.some((step) => step.status === "pending" || step.status === "in_progress");\n  }''',
'''  async loadForRoot(sessionId: string, rootMessageID: string): Promise<TaskPlan | null> {\n    if (!rootMessageID.trim()) return null;\n    const plan = await this.load(sessionId);\n    if (!plan || plan.rootMessageID !== rootMessageID || plan.taskEpoch !== rootMessageID) return null;\n    return plan;\n  }\n\n  async hasActionablePlan(sessionId: string, rootMessageID: string): Promise<boolean> {\n    const plan = await this.loadForRoot(sessionId, rootMessageID);\n    if (!plan) return false;\n    return plan.steps.some((step) => step.status === "pending" || step.status === "in_progress");\n  }''')

replace("distribution/plugins/src/task-state/gate.ts",
'''export async function requirePlan(store: TaskStateStore, sessionId: string, operation: string): Promise<void> {\n  try {\n    if (await store.hasActionablePlan(sessionId)) return;''',
'''export async function requirePlan(store: TaskStateStore, sessionId: string, rootMessageID: string, operation: string): Promise<void> {\n  try {\n    if (rootMessageID.trim() && await store.hasActionablePlan(sessionId, rootMessageID)) return;''')

replace("distribution/plugins/src/tools/types.ts",
'''  sessionId: string;\n  agent: string;''',
'''  sessionId: string;\n  rootTurnId: string;\n  agent: string;''')

replace("distribution/plugins/src/tools/task-plan.ts",
'''        const plan = await store.create(ctx.sessionId, input.goal, input.steps);''',
'''        if (!ctx.rootTurnId?.trim()) throw new ToolError("PLAN_REQUIRED", "task_plan requires a current root turn");\n        const plan = await store.create(ctx.sessionId, input.goal, input.steps, ctx.rootTurnId);''')
replace("distribution/plugins/src/tools/task-step.ts",
'''        const plan = await store.updateStep(ctx.sessionId, input.stepId, input.status, input.evidence);''',
'''        if (!ctx.rootTurnId?.trim()) throw new ToolError("PLAN_REQUIRED", "task_step requires a current root turn");\n        const plan = await store.updateStep(ctx.sessionId, input.stepId, input.status, input.evidence, ctx.rootTurnId);''')
replace("distribution/plugins/src/tools/task-status.ts",
'''        const plan = await store.load(ctx.sessionId);''',
'''        if (!ctx.rootTurnId?.trim()) {\n          ctx.guard.after({\n            sessionId: ctx.sessionId, agent: ctx.agent, tool: "task_status", action: "plan",\n            projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: true,\n          });\n          return ok({ plan: null });\n        }\n        const plan = await store.loadForRoot(ctx.sessionId, ctx.rootTurnId);''')

# OpenCode local SDK mirror gets the authoritative v1.18.11 chat.message hook shape.
replace("distribution/plugins/src/opencode/types.ts",
'''export type PluginOptions = Record<string, unknown>;''',
'''export type ChatUserMessage = {\n  id?: string;\n  sessionID?: string;\n  role?: string;\n  [key: string]: unknown;\n};\n\nexport type ChatPart = {\n  type?: string;\n  synthetic?: boolean;\n  metadata?: { [key: string]: unknown };\n  [key: string]: unknown;\n};\n\nexport type PluginOptions = Record<string, unknown>;''')
replace("distribution/plugins/src/opencode/types.ts",
'''export type Hooks = {\n  tool?: { [key: string]: ToolDefinition };''',
'''export type Hooks = {\n  tool?: { [key: string]: ToolDefinition };\n  "chat.message"?: (\n    input: {\n      sessionID: string;\n      agent?: string;\n      model?: { providerID: string; modelID: string };\n      messageID?: string;\n      variant?: string;\n    },\n    output: { message: ChatUserMessage; parts: ChatPart[] },\n  ) => Promise<void>;''')

# Root epoch tracker accepts only Codea verification-control metadata as a continuation override.
replace("distribution/plugins/src/task-state/epoch.ts",
'''    if (part.synthetic !== true) continue;\n    const metadata = part.metadata;''',
'''    const metadata = part.metadata;''')

# Plugin entry: derive current root only from chat.message, bind all task tools, exact-match mutation gate.
replace("distribution/plugins/src/opencode/entry.ts",
'''import { TaskStateStore } from "../task-state/store";\nimport { requirePlan } from "../task-state/gate";''',
'''import { TaskStateStore } from "../task-state/store";\nimport { RootTurnEpochs } from "../task-state/epoch";\nimport { requirePlan } from "../task-state/gate";''')
replace("distribution/plugins/src/opencode/entry.ts",
'''function toCodeaContext(octx: ToolContext, audit: AuditLogger, guard: RuntimeSecurityGuard, ownership?: WriteOwnership): CodeaToolContext {\n  return { sessionId: octx.sessionID, agent: octx.agent, projectRoot: octx.directory, audit, guard, ownership };\n}''',
'''function toCodeaContext(octx: ToolContext, audit: AuditLogger, guard: RuntimeSecurityGuard, rootTurnId: string, ownership?: WriteOwnership): CodeaToolContext {\n  return { sessionId: octx.sessionID, rootTurnId, agent: octx.agent, projectRoot: octx.directory, audit, guard, ownership };\n}''')
replace("distribution/plugins/src/opencode/entry.ts",
'''  sessionId: string,\n  agent: string,\n  tool: string,''',
'''  sessionId: string,\n  rootTurnId: string,\n  agent: string,\n  tool: string,''')
replace("distribution/plugins/src/opencode/entry.ts",
'''    await requirePlan(taskState, sessionId, `${tool}:${action}`);''',
'''    await requirePlan(taskState, sessionId, rootTurnId, `${tool}:${action}`);''')
replace("distribution/plugins/src/opencode/entry.ts",
'''  taskState: TaskStateStore,\n  ownershipFactory?: OwnershipFactory,\n): ToolDefinition {''',
'''  taskState: TaskStateStore,\n  rootTurns: RootTurnEpochs,\n  ownershipFactory?: OwnershipFactory,\n): ToolDefinition {''')
replace("distribution/plugins/src/opencode/entry.ts",
'''      const codeaCtx = toCodeaContext(octx, audit, guard, ownership);''',
'''      const rootTurnId = rootTurns.current(octx.sessionID);\n      const codeaCtx = toCodeaContext(octx, audit, guard, rootTurnId, ownership);''')
replace("distribution/plugins/src/opencode/entry.ts",
'''        await requirePlanForOperation(taskState, guard, octx.sessionID, octx.agent, name, action, octx.directory);''',
'''        await requirePlanForOperation(taskState, guard, octx.sessionID, rootTurnId, octx.agent, name, action, octx.directory);''')
replace("distribution/plugins/src/opencode/entry.ts",
'''    const taskState = new TaskStateStore({\n      workspaceRoot: input.directory,\n      codeaHome: process.env.CODEA_HOME || path.join(os.homedir(), ".codea"),\n    });''',
'''    const taskState = new TaskStateStore({\n      workspaceRoot: input.directory,\n      codeaHome: process.env.CODEA_HOME || path.join(os.homedir(), ".codea"),\n    });\n    const rootTurns = new RootTurnEpochs();''')
for name, extra in [
    ("collect_review_context", ""), ("analyze_test_project", ""),
    ("write_test_file", ", ownershipFactory"), ("run_project_test", ""),
    ("extract_api_spec", ""), ("validate_api_example", ""), ("write_document", ""),
    ("task_plan", ""), ("task_step", ""), ("task_status", ""),
]:
    old = f'{name}: adaptTool("{name}", '
    # add rootTurns immediately before optional ownership at the end using exact line replacements below
replace("distribution/plugins/src/opencode/entry.ts",
'''      collect_review_context: adaptTool("collect_review_context", collectReviewContextTool, audit, guard, taskState),\n      analyze_test_project: adaptTool("analyze_test_project", analyzeTestProjectTool, audit, guard, taskState),\n      write_test_file: adaptTool("write_test_file", writeTestFileTool, audit, guard, taskState, ownershipFactory),\n      run_project_test: adaptTool("run_project_test", runProjectTestTool, audit, guard, taskState),\n      extract_api_spec: adaptTool("extract_api_spec", extractApiSpecTool, audit, guard, taskState),\n      validate_api_example: adaptTool("validate_api_example", validateApiExampleTool, audit, guard, taskState),\n      write_document: adaptTool("write_document", writeDocumentTool, audit, guard, taskState),\n      task_plan: adaptTool("task_plan", createTaskPlanTool(taskState), audit, guard, taskState),\n      task_step: adaptTool("task_step", createTaskStepTool(taskState), audit, guard, taskState),\n      task_status: adaptTool("task_status", createTaskStatusTool(taskState), audit, guard, taskState),''',
'''      collect_review_context: adaptTool("collect_review_context", collectReviewContextTool, audit, guard, taskState, rootTurns),\n      analyze_test_project: adaptTool("analyze_test_project", analyzeTestProjectTool, audit, guard, taskState, rootTurns),\n      write_test_file: adaptTool("write_test_file", writeTestFileTool, audit, guard, taskState, rootTurns, ownershipFactory),\n      run_project_test: adaptTool("run_project_test", runProjectTestTool, audit, guard, taskState, rootTurns),\n      extract_api_spec: adaptTool("extract_api_spec", extractApiSpecTool, audit, guard, taskState, rootTurns),\n      validate_api_example: adaptTool("validate_api_example", validateApiExampleTool, audit, guard, taskState, rootTurns),\n      write_document: adaptTool("write_document", writeDocumentTool, audit, guard, taskState, rootTurns),\n      task_plan: adaptTool("task_plan", createTaskPlanTool(taskState), audit, guard, taskState, rootTurns),\n      task_step: adaptTool("task_step", createTaskStepTool(taskState), audit, guard, taskState, rootTurns),\n      task_status: adaptTool("task_status", createTaskStatusTool(taskState), audit, guard, taskState, rootTurns),''')
replace("distribution/plugins/src/opencode/entry.ts",
'''    const hooks: Hooks = {\n      tool: tools,''',
'''    const hooks: Hooks = {\n      tool: tools,\n      "chat.message": async (hookInput, output) => {\n        rootTurns.observe(hookInput, output);\n      },''')
replace("distribution/plugins/src/opencode/entry.ts",
'''          await requirePlanForOperation(taskState, guard, hookInput.sessionID, "", tool, "execute", input.directory);''',
'''          await requirePlanForOperation(taskState, guard, hookInput.sessionID, rootTurns.current(hookInput.sessionID), "", tool, "execute", input.directory);''')
replace("distribution/plugins/src/opencode/entry.ts",
'''          await requirePlanForOperation(taskState, guard, hookInput.sessionID, "", tool, "write", input.directory);''',
'''          await requirePlanForOperation(taskState, guard, hookInput.sessionID, rootTurns.current(hookInput.sessionID), "", tool, "write", input.directory);''')

# Test helper gives direct tool unit tests an explicit root; plugin tests must use chat.message.
replace("distribution/plugins/tests/helpers.ts",
'''    sessionId: "test-session",\n    agent: "unit-test-generator",''',
'''    sessionId: "test-session",\n    rootTurnId: "test-root",\n    agent: "unit-test-generator",''')
replace("distribution/plugins/tests/task-state-store.test.ts", "expect(plan.version).toBe(1);", "expect(plan.version).toBe(2);\n      expect(plan.rootMessageID).toBe(`session-${count}`);\n      expect(plan.taskEpoch).toBe(`session-${count}`);")
replace("distribution/plugins/tests/task-state-store.test.ts", 'expect(await store.hasActionablePlan("a")).toBe(true);', 'expect(await store.hasActionablePlan("a", "a")).toBe(true);')
replace("distribution/plugins/tests/task-planning-tools.test.ts",
'''      const hooks = await plugin.server(makePluginInput(root), { auditLog });\n      expect(Object.keys(hooks.tool ?? {})).toEqual(expect.arrayContaining(["task_plan", "task_step", "task_status"]));''',
'''      const hooks = await plugin.server(makePluginInput(root), { auditLog });\n      await hooks["chat.message"]!({ sessionID: "session-plugin", messageID: "U-plugin", agent: "general" }, { message: { id: "U-plugin", sessionID: "session-plugin", role: "user" }, parts: [] });\n      expect(Object.keys(hooks.tool ?? {})).toEqual(expect.arrayContaining(["task_plan", "task_step", "task_status"]));''')

# Codea runtime identity contract: map assistant parent IDs without leaking vendor DTOs.
replace("tui/internal/runtime/events.go",
'''\tMessageID       string\n\tPartID          string''',
'''\tMessageID       string\n\tParentMessageID string\n\tMessageRole     string\n\tPartID          string''')
replace("tui/internal/opencode/event_mapper.go",
'''type sseSessionInfo struct {\n\tProjectID string `json:"projectID"`\n\tID        string `json:"id"`\n}''',
'''type sseSessionInfo struct {\n\tProjectID string `json:"projectID"`\n\tID        string `json:"id"`\n\tSessionID string `json:"sessionID"`\n\tRole      string `json:"role"`\n\tParentID  string `json:"parentID"`\n}''')
replace("tui/internal/opencode/event_mapper.go",
'''\t\t\tif event.MessageID == "" && info.ID != "" && payload.Type == "message.updated" {\n\t\t\t\tevent.MessageID = info.ID\n\t\t\t}\n\t\t}\n\t}''',
'''\t\t\tif payload.Type == "message.updated" {\n\t\t\t\tif event.SessionID == "" && info.SessionID != "" { event.SessionID = info.SessionID }\n\t\t\t\tif event.MessageID == "" && info.ID != "" { event.MessageID = info.ID }\n\t\t\t\tevent.MessageRole = strings.TrimSpace(info.Role)\n\t\t\t\tevent.ParentMessageID = strings.TrimSpace(info.ParentID)\n\t\t\t}\n\t\t}\n\t}''')

# Application owns assistant-message -> root-turn mapping and keeps MessageID isolation fail-closed.
replace("tui/internal/app/model.go",
'''\ttaskExecution           TaskExecutionState\n\tviewMode                ViewMode\n\tactiveTurnID            string''',
'''\ttaskExecution           TaskExecutionState\n\tviewMode                ViewMode\n\tactiveTurnID            string\n\tmessageRootTurns        map[string]string''')
replace("tui/internal/app/model.go",
'''\t\texecutionTrace:  newExecutionTrace(),\n\t\tviewMode:        ViewNormal,''',
'''\t\texecutionTrace:   newExecutionTrace(),\n\t\tviewMode:         ViewNormal,\n\t\tmessageRootTurns: make(map[string]string),''')
replace("tui/internal/app/execution_trace.go",
'''\tm.activeTurnID = req.MessageID\n\tm.resetTaskExecution(req.MessageID)''',
'''\tm.activeTurnID = req.MessageID\n\tm.resetTaskExecution(req.MessageID)\n\tm.recordMessageRoot(req.MessageID, req.MessageID)''')
replace("tui/internal/app/execution_trace.go",
'''\tentry := ExecutionTraceEntry{\n\t\tCategory: TraceTool, Title: safeTraceText(ev.Tool.Name), Detail: safeTraceDetail(ev), Status: status,\n\t\tInvocationKey: key, SessionID: runtime.SessionID(ev.SessionID), TurnID: m.activeTurnID,\n\t}''',
'''\tentry := ExecutionTraceEntry{\n\t\tCategory: TraceTool, Title: safeTraceText(ev.Tool.Name), Detail: safeTraceDetail(ev), Status: status,\n\t\tInvocationKey: key, SessionID: runtime.SessionID(ev.SessionID), TurnID: m.eventRootTurnID(ev),\n\t}''')
replace("tui/internal/app/task_execution.go",
'''func (m *Model) resetTaskExecution(turnID string) {\n\tm.taskExecution = TaskExecutionState{RootTurnID: strings.TrimSpace(turnID)}\n}\n\nfunc (m *Model) observeTaskExecutionEvent(ev runtime.Event) {\n\tif m.activeTurnID == "" {''',
'''func (m *Model) resetTaskExecution(turnID string) {\n\tm.taskExecution = TaskExecutionState{RootTurnID: strings.TrimSpace(turnID)}\n}\n\nfunc messageRootKey(sessionID runtime.SessionID, messageID string) string {\n\treturn string(sessionID) + "\\x00" + strings.TrimSpace(messageID)\n}\n\nfunc (m *Model) recordMessageRoot(messageID, rootTurnID string) {\n\tmessageID = strings.TrimSpace(messageID)\n\trootTurnID = strings.TrimSpace(rootTurnID)\n\tif messageID == "" || rootTurnID == "" { return }\n\tif m.messageRootTurns == nil { m.messageRootTurns = make(map[string]string) }\n\tm.messageRootTurns[messageRootKey(m.sessionID, messageID)] = rootTurnID\n}\n\nfunc (m *Model) rootTurnForMessage(messageID string) string {\n\tmessageID = strings.TrimSpace(messageID)\n\tif messageID == "" { return "" }\n\tif root := m.messageRootTurns[messageRootKey(m.sessionID, messageID)]; root != "" { return root }\n\treturn ""\n}\n\nfunc (m *Model) observeMessageRoot(ev runtime.Event) {\n\tif ev.Type != "message.updated" || strings.TrimSpace(ev.MessageID) == "" { return }\n\tif ev.SessionID != "" && m.sessionID != "" && runtime.SessionID(ev.SessionID) != m.sessionID { return }\n\tif strings.TrimSpace(ev.MessageRole) != "assistant" { return }\n\tparent := strings.TrimSpace(ev.ParentMessageID)\n\tif parent == "" { return }\n\troot := m.rootTurnForMessage(parent)\n\tif root == "" && parent == m.activeTurnID { root = parent }\n\tif root != "" { m.recordMessageRoot(ev.MessageID, root) }\n}\n\nfunc (m *Model) eventRootTurnID(ev runtime.Event) string {\n\tif strings.TrimSpace(ev.MessageID) == "" { return m.activeTurnID }\n\treturn m.rootTurnForMessage(ev.MessageID)\n}\n\nfunc (m *Model) observeTaskExecutionEvent(ev runtime.Event) {\n\tm.observeMessageRoot(ev)\n\tif m.activeTurnID == "" {''')
replace("tui/internal/app/task_execution.go",
'''\tif ev.MessageID != "" && ev.MessageID != m.taskExecution.RootTurnID {\n\t\treturn\n\t}\n\tif ev.Tool == nil {''',
'''\tif ev.MessageID != "" {\n\t\troot := m.eventRootTurnID(ev)\n\t\tif root == "" || root != m.taskExecution.RootTurnID { return }\n\t}\n\tif ev.Tool == nil {''')
replace("tui/internal/app/update.go",
'''\tm.executionTrace.Reset()\n\tm.activeTurnID = ""''',
'''\tm.executionTrace.Reset()\n\tm.activeTurnID = ""\n\tm.messageRootTurns = make(map[string]string)''')
# update.go contains the same reset sequence twice (resume + clear); replace() changed all occurrences.

# Formal marker script must include the two new epoch regressions.
replace("tests/task29-agent-planning.sh",
'''pushd "$ROOT/distribution/plugins" >/dev/null\nbun test tests/task29-acceptance.test.ts''',
'''pushd "$ROOT/distribution/plugins" >/dev/null\nbun test tests/task29-acceptance.test.ts tests/task-root-epoch.test.ts''')

# Windows/macOS focused Gate includes mapper identity regression.
replace(".github/workflows/task29-gates.yml",
'''-run 'TaskExecution|TaskPlan|ToolMetadata|Trace|Session|Professional|Agent|Materialize|TaskStrategy' -count=1''',
'''-run 'TaskExecution|TaskPlan|Task29MessageUpdated|ToolMetadata|Trace|Session|Professional|Agent|Materialize|TaskStrategy' -count=1''')

# Export epoch contract for focused unit use; no Task30 implementation added.
replace("distribution/plugins/src/index.ts",
'''export * from "./task-state/gate";''',
'''export * from "./task-state/gate";\nexport * from "./task-state/epoch";''')

print("Task 29 root epoch patch applied")
