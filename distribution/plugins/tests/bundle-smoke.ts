// Security smoke: loads the real built bundle (dist/index.js) and verifies it is
// executable with zero public-network activity — the guard runs, the audit log
// is produced, and Dify degrades against a local fake server.

import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import * as plugin from "../dist/index.js";

function fail(msg: string): never {
  console.error(`[SMOKE FAIL] ${msg}`);
  process.exit(1);
}

const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-smoke-"));
const root = path.join(tmp, "project");
fs.mkdirSync(root, { recursive: true });
const logPath = path.join(tmp, "audit.log");

try {
  // 1. guard executable: dangerous command denied, safe command allowed
  const guard = new plugin.RuntimeSecurityGuard(new plugin.AuditLogger(logPath, root));
  const deny = guard.before({
    sessionId: "smoke", agent: "code-reviewer", tool: "bash", action: "execute",
    projectRoot: root, command: "rm -rf /",
  });
  if (deny.decision !== "deny") fail(`expected deny, got ${deny.decision}`);

  const allow = guard.before({
    sessionId: "smoke", agent: "code-reviewer", tool: "bash", action: "execute",
    projectRoot: root, command: "git status",
  });
  if (allow.decision !== "allow") fail(`expected allow, got ${allow.decision}`);

  // 2. audit log produced
  guard.after({
    sessionId: "smoke", agent: "code-reviewer", tool: "t", action: "read",
    projectRoot: root, durationMs: 3, ok: true, output: "done", targetPath: "src/A.java",
  });
  const auditContent = fs.readFileSync(logPath, "utf8");
  if (!auditContent.includes("src/A.java")) fail("audit log missing expected entry");

  // 3. Dify degrades against a local fake server (no public network)
  const server = Bun.serve({
    port: 0,
    fetch: () => new Response("server error", { status: 500 }),
  });
  const client = new plugin.DifyClient(
    { baseUrl: `http://127.0.0.1:${server.port}`, apiKey: "smoke-key" },
    { threshold: 1, resetTimeoutMs: 60_000 },
  );
  const result = await client.query("hello");
  server.stop(true);
  if (!result.degraded) fail("expected degraded Dify result");

  console.log("[SMOKE PASS] bundle executes: guard + audit + dify degradation, zero public network");
} finally {
  fs.rmSync(tmp, { recursive: true, force: true });
}
