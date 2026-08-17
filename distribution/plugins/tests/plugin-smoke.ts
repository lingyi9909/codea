// Real OpenCode plugin adapter smoke. Loads the built bundle's DEFAULT export
// (the exact contract OpenCode v1.18.11 reads via readV1Plugin → mod.default),
// invokes server() to obtain the Hooks.tool map (the exact contract the tool
// registry iterates via fromPlugin), then drives each integration point with a
// mock of OpenCode's plugin ToolContext:
//   - 8 tools registered (7 enterprise + dify-query)
//   - path policy deny → execute throws (adapter aborts)
//   - DLP input deny → execute throws
//   - write action → ctx.ask (enters permission flow)
//   - output DLP blocks a secret in a tool result
//   - dify-query degrades when Dify is not configured
// Zero public network; only local git + fs.

import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";

function fail(msg: string): never {
  console.error(`[PLUGIN SMOKE FAIL] ${msg}`);
  process.exit(1);
}

const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-plugin-smoke-"));
const root = path.join(tmp, "project");
fs.mkdirSync(root, { recursive: true });
const logPath = path.join(tmp, "audit.log");

type Ask = { permission: string; patterns: string[]; always: string[]; metadata: Record<string, unknown> };

function makeToolCtx(asks: Ask[]) {
  return {
    sessionID: "smoke-session",
    messageID: "smoke-message",
    agent: "unit-test-generator",
    directory: root,
    worktree: root,
    abort: new AbortController().signal,
    metadata() {},
    ask: async (req: Ask) => {
      asks.push(req);
    },
  };
}

try {
  const mod = await import("../dist/index.js");
  const pluginModule = mod.default;

  // 1. default export contract (id + server) — what readV1Plugin checks.
  if (!pluginModule || typeof pluginModule !== "object") fail("bundle has no default export");
  if (pluginModule.id !== "codea-enterprise") fail(`unexpected plugin id: ${pluginModule.id}`);
  if (typeof pluginModule.server !== "function") fail("plugin default export has no server()");

  // 2. server() → Hooks.tool with all 8 tools.
  const input = {
    client: {},
    project: {},
    directory: root,
    worktree: root,
    experimental_workspace: { register() {} },
    serverUrl: new URL("http://127.0.0.1:4096"),
    $: {},
  };
  const hooks = await pluginModule.server(input, { auditLog: logPath });
  const tools = hooks.tool ?? {};
  const expected = [
    "collect_review_context",
    "analyze_test_project",
    "write_test_file",
    "run_project_test",
    "extract_api_spec",
    "validate_api_example",
    "write_document",
    "dify-query",
  ];
  for (const name of expected) {
    if (!tools[name] || typeof tools[name].execute !== "function") {
      fail(`tool not registered: ${name}`);
    }
  }

  // 3. path policy deny → adapter aborts with path-violation.
  const asks: Ask[] = [];
  let threw = "";
  try {
    await tools.write_test_file.execute({ path: "../../etc/passwd", content: "x" }, makeToolCtx(asks));
  } catch (e) {
    threw = (e as Error).message ?? "";
  }
  if (!threw.includes("path-violation")) fail(`expected path-violation deny, got: ${threw || "<no throw>"}`);

  // 4. DLP input deny → adapter aborts with dlp-blocked.
  threw = "";
  try {
    await tools.write_test_file.execute(
      { path: "src/test/java/FooTest.java", content: "password=hunter2" },
      makeToolCtx(asks),
    );
  } catch (e) {
    threw = (e as Error).message ?? "";
  }
  if (!threw.includes("dlp-blocked")) fail(`expected dlp-blocked deny, got: ${threw || "<no throw>"}`);

  // 5. write action → ctx.ask enters the permission flow, then the file is written.
  fs.mkdirSync(path.join(root, "src", "test", "java"), { recursive: true });
  const writeAsks: Ask[] = [];
  const writeRes = await tools.write_test_file.execute(
    { path: "src/test/java/FooTest.java", content: "class FooTest {}\n" },
    makeToolCtx(writeAsks),
  );
  if (writeAsks.length !== 1 || writeAsks[0].permission !== "write_test_file") {
    fail(`expected one write_test_file permission ask, got ${writeAsks.length}`);
  }
  if (typeof writeRes !== "object" || writeRes.metadata?.ok !== true) {
    fail(`expected ok write result, got ${JSON.stringify(writeRes)}`);
  }
  if (!fs.existsSync(path.join(root, "src", "test", "java", "FooTest.java"))) {
    fail("write_test_file did not write the file");
  }

  // 6. output DLP blocks a secret that a read tool returns to the model.
  const reviewRoot = path.join(tmp, "review");
  fs.mkdirSync(reviewRoot, { recursive: true });
  const git = async (args: string[]) => {
    const p = Bun.spawnSync(["git", ...args], { cwd: reviewRoot });
    if (p.exitCode !== 0) fail(`git ${args.join(" ")} failed: ${p.stderr.toString()}`);
  };
  await git(["init", "-q"]);
  await git(["config", "user.email", "t@example.com"]);
  await git(["config", "user.name", "t"]);
  fs.writeFileSync(path.join(reviewRoot, "notes.txt"), "hello\n");
  await git(["add", "."]);
  await git(["commit", "-qm", "init"]);
  fs.writeFileSync(path.join(reviewRoot, "notes.txt"), "hello\npassword=supersecret\n");

  const reviewCtx = { ...makeToolCtx([]), directory: reviewRoot, worktree: reviewRoot };
  const reviewRes = await tools.collect_review_context.execute({ source: "unstaged" }, reviewCtx);
  if (typeof reviewRes === "object" && reviewRes.metadata?.dlpBlocked !== true) {
    fail(`expected output DLP block on secret in diff, got ${JSON.stringify(reviewRes)}`);
  }
  if (typeof reviewRes === "object" && !String(reviewRes.output).includes("DLP blocked")) {
    fail(`expected blocked output marker, got ${JSON.stringify(reviewRes)}`);
  }

  // 7. dify-query degrades when Dify is not configured (no public network).
  const difyRes = await tools["dify-query"].execute({ question: "hello" }, makeToolCtx([]));
  if (typeof difyRes !== "object" || difyRes.metadata?.degraded !== true) {
    fail(`expected dify degraded, got ${JSON.stringify(difyRes)}`);
  }

  console.log("[PLUGIN SMOKE PASS] default export → server() → 8 tools: path deny, DLP deny, permission, output DLP, dify degraded");
} finally {
  fs.rmSync(tmp, { recursive: true, force: true });
}
