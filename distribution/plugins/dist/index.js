// @bun
// src/security/types.ts
var RiskSafe = "safe";
var RiskAsk = "ask";
var RiskDeny = "deny";
// src/security/command-policy.ts
var DANGEROUS_COMMANDS = new Set([
  "sudo",
  "doas",
  "su",
  "curl",
  "wget",
  "nc",
  "netcat",
  "ncat",
  "telnet",
  "rm",
  "rmdir",
  "del",
  "erase",
  "sh",
  "bash",
  "zsh",
  "cmd",
  "powershell",
  "pwsh",
  "remove-item",
  "invoke-webrequest",
  "invoke-expression",
  "start-process"
]);
var SAFE_COMMANDS = [
  /^git\s+status(\s+.*)?$/,
  /^git\s+diff(\s+.*)?$/,
  /^git\s+log(\s+.*)?$/,
  /^git\s+show(\s+.*)?$/,
  /^git\s+rev-parse(\s+.*)?$/,
  /^git\s+branch(\s+.*)?$/,
  /^git\s+ls-files(\s+.*)?$/,
  /^git\s+shortlog(\s+.*)?$/,
  /^ls(\s+.*)?$/,
  /^pwd$/,
  /^cat(\s+.*)?$/,
  /^head(\s+.*)?$/,
  /^tail(\s+.*)?$/,
  /^grep(\s+.*)?$/,
  /^find(\s+.*)?$/
];
function tokenize(input) {
  return input.toLowerCase().split(/[\s;&|()<>$`"'\\]+/).filter((t) => t.length > 0);
}
function analyzeCommand(input) {
  const command = input.trim();
  const analysis = {
    risk: RiskAsk,
    command,
    hasPipe: false,
    hasRedirect: false,
    hasSubCmd: false,
    hasChain: false,
    matchedRule: ""
  };
  analysis.hasPipe = /\|(?!\|)/.test(command);
  analysis.hasRedirect = />|>>|</.test(command);
  analysis.hasSubCmd = /\$\(/.test(command) || /`/.test(command);
  analysis.hasChain = /&&|\|\||;/.test(command);
  for (const token of tokenize(command)) {
    if (DANGEROUS_COMMANDS.has(token)) {
      analysis.risk = RiskDeny;
      analysis.matchedRule = token;
      return analysis;
    }
  }
  if (analysis.hasPipe || analysis.hasRedirect || analysis.hasSubCmd || analysis.hasChain) {
    analysis.risk = RiskAsk;
    return analysis;
  }
  const lower = command.toLowerCase();
  for (const re of SAFE_COMMANDS) {
    if (re.test(lower)) {
      analysis.risk = RiskSafe;
      return analysis;
    }
  }
  analysis.risk = RiskAsk;
  return analysis;
}
function isDangerous(input) {
  return analyzeCommand(input).risk === RiskDeny;
}
function riskLabel(risk) {
  return risk;
}
// src/security/dlp.ts
var MASK_SECRET = "[REDACTED]";
var MASK_PATH = "[REDACTED-PATH]";
var DLP_RULES = [
  { name: "authorization-header", severity: "high", action: "block", mask: MASK_SECRET, pattern: /authorization\s*:\s*bearer\s+[^\s,;'"]+/gi },
  { name: "bearer-token", severity: "high", action: "block", mask: MASK_SECRET, pattern: /bearer\s+[A-Za-z0-9\-._~+/]{16,}/gi },
  { name: "private-key", severity: "high", action: "block", mask: MASK_SECRET, pattern: /-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----/g },
  { name: "github-pat", severity: "high", action: "block", mask: MASK_SECRET, pattern: /\bghp_[A-Za-z0-9]{36,}\b/g },
  { name: "github-fine-pat", severity: "high", action: "block", mask: MASK_SECRET, pattern: /\bgithub_pat_[A-Za-z0-9_]{40,}\b/g },
  { name: "aws-access-key", severity: "high", action: "block", mask: MASK_SECRET, pattern: /\bAKIA[0-9A-Z]{16}\b/g },
  { name: "api-key", severity: "high", action: "block", mask: MASK_SECRET, pattern: /\b(api[_-]?key|apikey|access[_-]?token|secret[_-]?key|client[_-]?secret)\b\s*[:=]\s*["']?[^\s,;'"&]+/gi },
  { name: "password", severity: "high", action: "block", mask: MASK_SECRET, pattern: /\bpassword\b\s*[:=]\s*["']?[^\s,;'"&]+/gi },
  { name: "passwd", severity: "high", action: "block", mask: MASK_SECRET, pattern: /\bpasswd\b\s*[:=]\s*["']?[^\s,;'"&]+/gi },
  { name: "token", severity: "high", action: "block", mask: MASK_SECRET, pattern: /\btoken\b\s*[:=]\s*["']?[^\s,;'"&]+/gi },
  { name: "secret-value", severity: "high", action: "block", mask: MASK_SECRET, pattern: /\b(secret|credential)\b\s*[:=]\s*["']?[^\s,;'"&]+/gi },
  { name: "env-file", severity: "medium", action: "redact", mask: MASK_PATH, pattern: /\.env(?:\.\w+)?\b/g },
  { name: "ssh-key", severity: "medium", action: "redact", mask: MASK_PATH, pattern: /\bid_(?:rsa|ed25519|ecdsa|dsa)\b/g },
  { name: "pem-file", severity: "medium", action: "redact", mask: MASK_PATH, pattern: /[\w./-]+\.pem\b/g },
  { name: "credentials-file", severity: "medium", action: "redact", mask: MASK_PATH, pattern: /\bcredentials(?:\.\w+)?\b/gi }
];
function maskFor(rule) {
  return rule.mask;
}
function scanDlp(input, _context) {
  if (typeof input !== "string" || input.length === 0) {
    return { allowed: true, redacted: input, findings: [] };
  }
  const findings = [];
  let redacted = input;
  let allowed = true;
  for (const rule of DLP_RULES) {
    const matches = [...input.matchAll(rule.pattern)];
    if (matches.length === 0)
      continue;
    for (const m of matches) {
      findings.push({
        rule: rule.name,
        severity: rule.severity,
        start: m.index,
        end: m.index + m[0].length
      });
    }
    if (rule.action === "block") {
      allowed = false;
    }
    redacted = redacted.replace(rule.pattern, maskFor(rule));
  }
  return { allowed, redacted, findings };
}
function redact(input, context) {
  return scanDlp(input, context).redacted;
}
function hasBlockingSecret(input, context) {
  return !scanDlp(input, context).allowed;
}
// src/security/path-policy.ts
import * as fs from "fs";
import * as path from "path";

class PathViolationError extends Error {
  constructor(message) {
    super(message);
    this.name = "PathViolationError";
  }
}
var WINDOWS_DRIVE = /^[a-zA-Z]:[\\/]/;
var WINDOWS_UNC = /^\\\\/;
function normalizeSeparators(p) {
  return p.replace(/\\/g, "/");
}
function isWithin(root, target) {
  if (target === root)
    return true;
  const prefix = root.endsWith(path.sep) ? root : root + path.sep;
  return target.startsWith(prefix);
}
function realpathExisting(p) {
  let cur = path.resolve(p);
  const suffix = [];
  while (!fs.existsSync(cur)) {
    const parent = path.dirname(cur);
    if (parent === cur)
      break;
    suffix.unshift(path.basename(cur));
    cur = parent;
  }
  let real = fs.realpathSync(cur);
  for (const seg of suffix) {
    real = path.join(real, seg);
  }
  return real;
}
function resolveProjectPath(root, inputPath) {
  if (typeof inputPath !== "string" || inputPath.length === 0) {
    throw new PathViolationError("empty path");
  }
  if (WINDOWS_DRIVE.test(inputPath) || WINDOWS_UNC.test(inputPath)) {
    throw new PathViolationError("windows absolute or UNC path rejected");
  }
  const rootAbs = path.resolve(root);
  const target = path.resolve(rootAbs, normalizeSeparators(inputPath));
  if (!isWithin(rootAbs, target)) {
    throw new PathViolationError("path escapes project root");
  }
  let realRoot;
  let realTarget;
  try {
    realRoot = fs.realpathSync(rootAbs);
    realTarget = realpathExisting(target);
  } catch (err) {
    throw new PathViolationError(`cannot resolve path: ${err.message}`);
  }
  if (!isWithin(realRoot, realTarget)) {
    throw new PathViolationError("symlink escape detected");
  }
  return target;
}
function assertWithinRoot(root, p) {
  resolveProjectPath(root, p);
}
function assertWithinAllowedRoots(p, roots) {
  const target = path.resolve(p);
  for (const root of roots) {
    const rootAbs = path.resolve(root);
    if (!isWithin(rootAbs, target))
      continue;
    try {
      const realRoot = realpathExisting(rootAbs);
      const realTarget = realpathExisting(target);
      if (isWithin(realRoot, realTarget))
        return;
    } catch {}
  }
  throw new PathViolationError("path not within any allowed root");
}
function toRelativePath(root, p) {
  const rootAbs = path.resolve(root);
  const target = path.resolve(rootAbs, normalizeSeparators(p));
  const rel = path.relative(rootAbs, target);
  return normalizeSeparators(rel === "" ? "." : rel);
}
// src/dify-query.ts
class CircuitOpenError extends Error {
  constructor() {
    super("circuit open");
    this.name = "CircuitOpenError";
  }
}

class DifyHttpError extends Error {
  status;
  constructor(status) {
    super(`dify http ${status}`);
    this.status = status;
    this.name = "DifyHttpError";
  }
}

class DifyInvalidResponseError extends Error {
  constructor() {
    super("dify invalid response");
    this.name = "DifyInvalidResponseError";
  }
}

class CircuitBreaker {
  state = "closed";
  consecutiveFailures = 0;
  openedAt = 0;
  halfOpenProbeInFlight = false;
  opts;
  constructor(opts = {}) {
    this.opts = {
      threshold: opts.threshold ?? 3,
      resetTimeoutMs: opts.resetTimeoutMs ?? 60000,
      now: opts.now ?? (() => Date.now())
    };
  }
  get currentState() {
    this.tryTransitionToHalfOpen();
    return this.state;
  }
  get now() {
    return this.opts.now();
  }
  tryTransitionToHalfOpen() {
    if (this.state === "open" && this.now - this.openedAt >= this.opts.resetTimeoutMs) {
      this.state = "half-open";
      this.halfOpenProbeInFlight = false;
    }
  }
  async execute(fn) {
    this.tryTransitionToHalfOpen();
    if (this.state === "open") {
      throw new CircuitOpenError;
    }
    if (this.state === "half-open" && this.halfOpenProbeInFlight) {
      throw new CircuitOpenError;
    }
    if (this.state === "half-open") {
      this.halfOpenProbeInFlight = true;
    }
    try {
      const value = await fn();
      this.onSuccess();
      return value;
    } catch (err) {
      this.onFailure();
      throw err;
    }
  }
  onSuccess() {
    this.state = "closed";
    this.consecutiveFailures = 0;
    this.halfOpenProbeInFlight = false;
  }
  onFailure() {
    this.consecutiveFailures += 1;
    this.halfOpenProbeInFlight = false;
    if (this.state === "half-open" || this.consecutiveFailures >= this.opts.threshold) {
      this.state = "open";
      this.openedAt = this.now;
    }
  }
}

class DifyClient {
  breaker;
  config;
  constructor(config, breakerOpts) {
    this.config = config;
    this.breaker = new CircuitBreaker(breakerOpts);
  }
  get circuitState() {
    return this.breaker.currentState;
  }
  async query(question) {
    if (!this.config.baseUrl || !this.config.apiKey) {
      return { degraded: true, error: "dify-not-configured" };
    }
    try {
      const answer = await this.breaker.execute(() => this.fetchQuery(question));
      return { degraded: false, answer };
    } catch (err) {
      if (err instanceof CircuitOpenError) {
        return { degraded: true, error: "circuit-open" };
      }
      return { degraded: true, error: classifyDifyError(err) };
    }
  }
  async fetchQuery(question) {
    const timeoutMs = this.config.timeoutMs ?? 1e4;
    const controller = new AbortController;
    const timer = setTimeout(() => controller.abort(), timeoutMs);
    try {
      const res = await fetch(`${this.config.baseUrl}/chat-messages`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${this.config.apiKey}`
        },
        body: JSON.stringify({
          inputs: {},
          query: question,
          response_mode: "blocking",
          user: "codea"
        }),
        signal: controller.signal
      });
      if (!res.ok) {
        throw new DifyHttpError(res.status);
      }
      let data;
      try {
        data = await res.json();
      } catch {
        throw new DifyInvalidResponseError;
      }
      const answer = data?.answer;
      if (typeof answer !== "string") {
        throw new DifyInvalidResponseError;
      }
      return answer;
    } finally {
      clearTimeout(timer);
    }
  }
}
function classifyDifyError(err) {
  const name = err?.name ?? "";
  if (name === "AbortError" || name === "TimeoutError")
    return "timeout";
  if (err instanceof DifyHttpError) {
    return err.status >= 500 ? "http-5xx" : "http-4xx";
  }
  if (err instanceof DifyInvalidResponseError)
    return "invalid-response";
  return "network-error";
}
function difyConfigFromEnv(env = process.env) {
  const baseUrl = env["DIFY_BASE_URL"];
  const apiKey = env["DIFY_API_KEY"];
  if (!baseUrl || !apiKey)
    return null;
  return { baseUrl, apiKey };
}
// src/audit-log.ts
import * as fs2 from "fs";
import * as path2 from "path";
class AuditLogger {
  logPath;
  projectRoot;
  constructor(logPath, projectRoot) {
    this.logPath = logPath;
    this.projectRoot = projectRoot;
  }
  log(entry) {
    try {
      const sanitized = this.sanitize(entry);
      fs2.appendFileSync(this.logPath, JSON.stringify(sanitized) + `
`);
      return { ok: true };
    } catch (err) {
      return { ok: false, error: `audit-write-failed: ${err.message}` };
    }
  }
  sanitize(entry) {
    const out = {
      timestamp: entry.timestamp,
      sessionId: redact(entry.sessionId, "audit"),
      agent: entry.agent,
      tool: entry.tool,
      action: entry.action,
      result: entry.result,
      duration: entry.duration,
      errorCategory: entry.errorCategory
    };
    if (entry.relativePath) {
      out.relativePath = this.toProjectRelative(entry.relativePath);
    }
    return out;
  }
  toProjectRelative(p) {
    const rootAbs = path2.resolve(this.projectRoot);
    const target = path2.resolve(rootAbs, p);
    return path2.relative(rootAbs, target).replace(/\\/g, "/");
  }
}
// src/runtime-security-guard.ts
var WRITE_ACTIONS = new Set(["write", "create", "overwrite", "append", "edit"]);
function stringify(value) {
  if (typeof value === "string")
    return value;
  if (value === undefined || value === null)
    return "";
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

class RuntimeSecurityGuard {
  audit;
  constructor(audit) {
    this.audit = audit;
  }
  before(input) {
    if (input.targetPath && WRITE_ACTIONS.has(input.action)) {
      try {
        resolveProjectPath(input.projectRoot, input.targetPath);
      } catch (err) {
        this.auditDeny(input, `path-violation: ${err.message}`);
        return { decision: "deny", reason: `path-violation: ${err.message}` };
      }
    }
    if (input.command !== undefined) {
      const analysis = analyzeCommand(input.command);
      if (analysis.risk === RiskDeny) {
        this.auditDeny(input, `command-denied: ${analysis.matchedRule}`);
        return { decision: "deny", reason: `command-denied: ${analysis.matchedRule}` };
      }
      if (analysis.risk === RiskSafe) {
        return { decision: "allow" };
      }
      this.auditDeny(input, "command-requires-approval");
      return { decision: "ask", reason: "command-requires-approval" };
    }
    const dlp = scanDlp(stringify(input.input), "tool-input");
    if (!dlp.allowed) {
      const rule = dlp.findings[0]?.rule ?? "secret";
      this.auditDeny(input, `dlp-blocked: ${rule}`);
      return { decision: "deny", reason: `dlp-blocked: ${rule}` };
    }
    return { decision: "allow", redactedInput: this.redactValue(input.input, "tool-input") };
  }
  after(input) {
    const dlp = scanDlp(stringify(input.output), "tool-output");
    const result = !input.ok ? "error" : dlp.allowed ? "success" : "dlp-blocked";
    const errorCategory = input.ok ? dlp.allowed ? input.errorCategory : "DLP_BLOCKED" : input.errorCategory;
    const entry = {
      timestamp: new Date().toISOString(),
      sessionId: input.sessionId,
      agent: input.agent,
      tool: input.tool,
      action: input.action,
      result,
      duration: input.durationMs,
      relativePath: input.targetPath,
      errorCategory
    };
    this.audit.log(entry);
  }
  auditDeny(input, reason) {
    const entry = {
      timestamp: new Date().toISOString(),
      sessionId: input.sessionId,
      agent: input.agent,
      tool: input.tool,
      action: input.action,
      result: "denied",
      duration: 0,
      relativePath: input.targetPath,
      errorCategory: reason
    };
    this.audit.log(entry);
  }
  redactValue(value, context) {
    if (typeof value === "string")
      return scanDlp(value, context).redacted;
    if (Array.isArray(value))
      return value.map((v) => this.redactValue(v, context));
    if (value && typeof value === "object") {
      const out = {};
      for (const [k, v] of Object.entries(value)) {
        out[k] = this.redactValue(v, context);
      }
      return out;
    }
    return value;
  }
}
// src/permissions.ts
var ENTERPRISE_AGENTS = ["code-reviewer", "unit-test-generator", "api-documentation"];
var NATIVE_WRITE_EXEC = ["write", "edit", "bash"];
var READ_ONLY = new Set(["read", "grep", "glob"]);
var CONTROLLED = new Set([
  "collect_review_context",
  "analyze_test_project",
  "write_test_file",
  "run_project_test",
  "extract_api_spec",
  "validate_api_example",
  "write_document",
  "dify-query"
]);
function validatePermissions(config) {
  const issues = [];
  if (!config || typeof config !== "object" || !config.agents) {
    return [{ agent: "", message: "missing agents map" }];
  }
  for (const name of ENTERPRISE_AGENTS) {
    const perms = config.agents[name];
    if (!perms) {
      issues.push({ agent: name, message: "missing agent permissions" });
      continue;
    }
    for (const tool of NATIVE_WRITE_EXEC) {
      if (perms[tool] !== "deny") {
        issues.push({ agent: name, message: `${tool} must be deny, got ${perms[tool] ?? "unset"}` });
      }
    }
    for (const [tool, level] of Object.entries(perms)) {
      if (level !== "allow")
        continue;
      if (READ_ONLY.has(tool) || CONTROLLED.has(tool))
        continue;
      issues.push({ agent: name, message: `uncontrolled allow on ${tool}` });
    }
  }
  const general = config.agents["general"];
  if (!general) {
    issues.push({ agent: "general", message: "missing general permissions" });
  } else {
    for (const tool of NATIVE_WRITE_EXEC) {
      if (general[tool] === "deny") {
        issues.push({ agent: "general", message: `${tool} must not be deny (native capability)` });
      }
    }
  }
  return issues;
}
// src/tools/types.ts
function ok(data) {
  return { ok: true, data };
}
function err(error) {
  return { ok: false, error };
}
// src/tools/errors.ts
class ToolError extends Error {
  category;
  cause;
  constructor(category, message, cause) {
    super(message);
    this.name = "ToolError";
    this.category = category;
    this.cause = cause;
  }
  toJSON() {
    return { category: this.category, message: this.message };
  }
}
function invalidInput(message) {
  return new ToolError("INVALID_INPUT", message);
}
function pathViolation(message) {
  return new ToolError("PATH_VIOLATION", message);
}
function permissionDenied(message) {
  return new ToolError("PERMISSION_DENIED", message);
}
function dlpBlocked(message) {
  return new ToolError("DLP_BLOCKED", message);
}
function timeoutError(message) {
  return new ToolError("TIMEOUT", message);
}
function commandFailed(message, cause) {
  return new ToolError("COMMAND_FAILED", message, cause);
}
function parseFailed(message) {
  return new ToolError("PARSE_FAILED", message);
}
function notSupported(message) {
  return new ToolError("NOT_SUPPORTED", message);
}
function internalError(message, cause) {
  return new ToolError("INTERNAL_ERROR", message, cause);
}
// src/tools/failure-classifier.ts
var TIMEOUT_SIGNALS = new Set(["SIGTERM", "SIGKILL", "ETIMEDOUT"]);
function classifyError(err2) {
  if (err2 instanceof ToolError)
    return err2.category;
  if (err2 instanceof PathViolationError)
    return "PATH_VIOLATION";
  if (err2 instanceof Error) {
    const anyErr = err2;
    if (anyErr.killed || anyErr.signal && TIMEOUT_SIGNALS.has(anyErr.signal) || anyErr.code === "ETIMEDOUT") {
      return "TIMEOUT";
    }
  }
  return "INTERNAL_ERROR";
}
function toToolError(err2, fallbackMessage) {
  if (err2 instanceof ToolError)
    return err2;
  if (err2 instanceof PathViolationError)
    return new ToolError("PATH_VIOLATION", err2.message, err2);
  if (err2 instanceof Error)
    return new ToolError(classifyError(err2), err2.message || fallbackMessage, err2);
  return new ToolError("INTERNAL_ERROR", fallbackMessage, err2);
}
function classifyCommandFailure(exitCode, timedOut) {
  if (timedOut)
    return "TIMEOUT";
  return "COMMAND_FAILED";
}
// src/tools/filesystem.ts
import * as fs3 from "fs";
import * as path3 from "path";
function resolveInRoot(projectRoot, relPath) {
  try {
    return resolveProjectPath(projectRoot, relPath);
  } catch (e) {
    if (e instanceof PathViolationError)
      throw pathViolation(e.message);
    throw e;
  }
}
function fileExists(projectRoot, relPath) {
  return fs3.existsSync(resolveInRoot(projectRoot, relPath));
}
function readTextFile(projectRoot, relPath) {
  const abs = resolveInRoot(projectRoot, relPath);
  try {
    return fs3.readFileSync(abs, "utf8");
  } catch (e) {
    throw internalError(`read failed: ${relPath}`, e);
  }
}
function listDir(dir) {
  try {
    return fs3.readdirSync(dir);
  } catch (e) {
    throw internalError(`readdir failed: ${dir}`, e);
  }
}
function writeFileAtomic(opts) {
  const abs = resolveInRoot(opts.projectRoot, opts.relPath);
  if (opts.allowedRoots && opts.allowedRoots.length > 0) {
    const absRoots = opts.allowedRoots.map((r) => path3.resolve(opts.projectRoot, r));
    try {
      assertWithinAllowedRoots(abs, absRoots);
    } catch (e) {
      if (e instanceof PathViolationError)
        throw pathViolation(e.message);
      throw e;
    }
  }
  if (fs3.existsSync(abs) && !opts.overwrite) {
    throw permissionDenied(`file exists and overwrite is not allowed: ${opts.relPath}`);
  }
  const dlp = scanDlp(opts.content, "file-write");
  if (!dlp.allowed) {
    const rule = dlp.findings[0]?.rule ?? "secret";
    throw dlpBlocked(`content blocked by DLP: ${rule}`);
  }
  const dir = path3.dirname(abs);
  fs3.mkdirSync(dir, { recursive: true });
  const tmp = path3.join(dir, `.codea-tmp-${process.pid}-${Date.now()}-${Math.random().toString(36).slice(2)}`);
  try {
    fs3.writeFileSync(tmp, opts.content, "utf8");
    fs3.renameSync(tmp, abs);
  } catch (e) {
    try {
      fs3.rmSync(tmp, { force: true });
    } catch {}
    throw internalError(`write failed: ${opts.relPath}`, e);
  }
  return { path: abs, bytes: Buffer.byteLength(opts.content, "utf8") };
}

// src/tools/exec.ts
import { execFile } from "child_process";
var DEFAULT_TIMEOUT_MS = 30000;
var DEFAULT_MAX_BUFFER = 10 * 1024 * 1024;
function displayCommand(argv) {
  return argv.map((a) => /\s/.test(a) ? JSON.stringify(a) : a).join(" ");
}
function execCommand(argv, opts) {
  const file = argv[0];
  if (!file)
    throw commandFailed("empty command argv");
  const args = argv.slice(1);
  const timeoutMs = opts.timeoutMs ?? DEFAULT_TIMEOUT_MS;
  return new Promise((resolve4) => {
    execFile(file, args, {
      cwd: opts.cwd,
      timeout: timeoutMs,
      maxBuffer: opts.maxBuffer ?? DEFAULT_MAX_BUFFER,
      encoding: "utf8",
      env: { ...process.env, ...opts.env ?? {} }
    }, (error, stdout, stderr) => {
      const out = String(stdout ?? "");
      const errOut = String(stderr ?? "");
      const timedOut = !!(error && error.killed);
      const exitCode = error ? error.code ?? 1 : 0;
      resolve4({
        exitCode: typeof exitCode === "number" ? exitCode : 1,
        stdout: out,
        stderr: errOut,
        timedOut,
        command: displayCommand(argv)
      });
    });
  });
}

// src/tools/schemas.ts
function typeMatches(value, type) {
  switch (type) {
    case "string":
      return typeof value === "string";
    case "number":
      return typeof value === "number";
    case "integer":
      return typeof value === "number" && Number.isInteger(value);
    case "boolean":
      return typeof value === "boolean";
    case "array":
      return Array.isArray(value);
    case "object":
      return typeof value === "object" && value !== null && !Array.isArray(value);
    case "null":
      return value === null;
    default:
      return true;
  }
}
function checkProperty(value, prop, path4, issues) {
  if (value === undefined)
    return;
  if (prop.type !== undefined) {
    const types = Array.isArray(prop.type) ? prop.type : [prop.type];
    if (!types.some((t) => typeMatches(value, t))) {
      issues.push({ path: path4, message: `expected ${types.join("|")}, got ${typeof value}` });
      return;
    }
  }
  if (prop.enum !== undefined && !prop.enum.some((e) => e === value)) {
    issues.push({ path: path4, message: `must be one of ${JSON.stringify(prop.enum)}` });
  }
  if (typeof value === "number") {
    if (prop.minimum !== undefined && value < prop.minimum) {
      issues.push({ path: path4, message: `must be >= ${prop.minimum}` });
    }
    if (prop.maximum !== undefined && value > prop.maximum) {
      issues.push({ path: path4, message: `must be <= ${prop.maximum}` });
    }
  }
  if (typeof value === "string") {
    if (prop.minLength !== undefined && value.length < prop.minLength) {
      issues.push({ path: path4, message: `length must be >= ${prop.minLength}` });
    }
    if (prop.maxLength !== undefined && value.length > prop.maxLength) {
      issues.push({ path: path4, message: `length must be <= ${prop.maxLength}` });
    }
  }
  if (Array.isArray(value) && prop.items) {
    value.forEach((item, i) => checkProperty(item, prop.items, `${path4}[${i}]`, issues));
  }
  if (value !== null && typeof value === "object" && !Array.isArray(value) && prop.properties) {
    const obj = value;
    for (const key of prop.required ?? []) {
      if (obj[key] === undefined) {
        issues.push({ path: `${path4}.${key}`, message: "required" });
      }
    }
    for (const [key, child] of Object.entries(obj)) {
      if (child === undefined)
        continue;
      if (prop.additionalProperties === false && !(key in prop.properties)) {
        issues.push({ path: `${path4}.${key}`, message: "not allowed" });
        continue;
      }
      const childSchema = prop.properties[key];
      if (childSchema)
        checkProperty(child, childSchema, `${path4}.${key}`, issues);
    }
  }
}
function validateSchema(schema, value) {
  const issues = [];
  if (schema.type === "object") {
    if (value === null || typeof value !== "object" || Array.isArray(value)) {
      return [{ path: "$", message: "expected object" }];
    }
    const obj = value;
    for (const key of schema.required ?? []) {
      if (obj[key] === undefined) {
        issues.push({ path: `$.${key}`, message: "required" });
      }
    }
    if (schema.additionalProperties === false) {
      for (const key of Object.keys(obj)) {
        if (!(key in (schema.properties ?? {}))) {
          issues.push({ path: `$.${key}`, message: "not allowed" });
        }
      }
    }
    for (const [key, child] of Object.entries(obj)) {
      if (child === undefined)
        continue;
      const childSchema = schema.properties?.[key];
      if (childSchema)
        checkProperty(child, childSchema, `$.${key}`, issues);
    }
  } else {
    checkProperty(value, { type: schema.type }, "$", issues);
  }
  return issues;
}

// src/tools/collect-review-context.ts
var MAX_DIFF_BYTES = 5 * 1024 * 1024;
var SCHEMA = {
  type: "object",
  properties: {
    source: { type: "string", enum: ["staged", "unstaged", "base-branch", "commit", "range", "file-path"] },
    baseBranch: { type: "string" },
    commit: { type: "string" },
    rangeFrom: { type: "string" },
    rangeTo: { type: "string" },
    filePath: { type: "string" }
  },
  required: ["source"],
  additionalProperties: false
};
function buildGitDiffCommand(params) {
  switch (params.source) {
    case "staged":
      return ["git", "diff", "--cached"];
    case "unstaged":
      return ["git", "diff"];
    case "base-branch":
      return ["git", "diff", params.baseBranch ?? "origin/main"];
    case "commit":
      return ["git", "diff", `${params.commit}^`, params.commit];
    case "range":
      return ["git", "diff", `${params.rangeFrom}..${params.rangeTo}`];
    case "file-path":
      return ["git", "diff", "--", params.filePath];
  }
}
function validateInput(params) {
  const issues = validateSchema(SCHEMA, params);
  if (issues.length > 0) {
    throw invalidInput(`invalid input: ${issues.map((i) => `${i.path} ${i.message}`).join("; ")}`);
  }
  const p = params;
  if (p.source === "commit" && !p.commit)
    throw invalidInput("commit is required for source=commit");
  if (p.source === "range" && (!p.rangeFrom || !p.rangeTo))
    throw invalidInput("rangeFrom and rangeTo are required for source=range");
  if (p.source === "file-path" && !p.filePath)
    throw invalidInput("filePath is required for source=file-path");
  return p;
}
function parseUnifiedDiff(diff) {
  const files = [];
  let linesAdded = 0;
  let linesRemoved = 0;
  let current = null;
  let currentHunk = null;
  for (const raw of diff.split(`
`)) {
    if (raw.startsWith("diff --git ")) {
      if (current)
        files.push(current);
      current = { path: "", status: "modified", hunks: [] };
      currentHunk = null;
      const m = /^diff --git a\/(.*?) b\/(.*)$/.exec(raw);
      if (m) {
        current.path = m[2] ?? "";
        current.oldPath = m[1] ?? "";
      }
      continue;
    }
    if (!current)
      continue;
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
          lines: []
        };
        current.hunks.push(currentHunk);
      }
      continue;
    }
    if (currentHunk) {
      currentHunk.lines.push(raw);
      if (raw.startsWith("+") && !raw.startsWith("+++"))
        linesAdded++;
      if (raw.startsWith("-") && !raw.startsWith("---"))
        linesRemoved++;
    }
  }
  if (current)
    files.push(current);
  return { filesChanged: files.length, linesAdded, linesRemoved, files };
}
var collectReviewContextTool = {
  name: "collect_review_context",
  description: "Collect git diff context for code review. Returns exact file paths, line numbers, and diff hunks.",
  parameters: SCHEMA,
  async execute(params, ctx) {
    const started = Date.now();
    try {
      const input = validateInput(params);
      if (input.source === "file-path") {
        resolveInRoot(ctx.projectRoot, input.filePath);
      }
      const argv = buildGitDiffCommand(input);
      const result = await execCommand(argv, { cwd: ctx.projectRoot, timeoutMs: 30000 });
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
  }
};
// src/tools/analyze-test-project.ts
var DEFAULT_SOURCE_ROOTS = ["src/main/java"];
function detectBuildSystem(root) {
  if (fileExists(root, "pom.xml"))
    return "maven";
  if (fileExists(root, "build.gradle") || fileExists(root, "build.gradle.kts"))
    return "gradle";
  return "unknown";
}
function detectTestFramework(root, buildSystem) {
  const candidates = buildSystem === "maven" ? ["pom.xml"] : ["build.gradle", "build.gradle.kts"];
  const dependencies = [];
  let text = "";
  for (const f of candidates) {
    if (fileExists(root, f)) {
      text += readTextFile(root, f);
    }
  }
  const framework = /junit-jupiter|junit\.jupiter|org\.junit\.jupiter/.test(text) ? "JUnit 5" : /junit/.test(text) ? "JUnit 4" : "unknown";
  if (/mockito/.test(text))
    dependencies.push("mockito");
  if (/assertj/.test(text))
    dependencies.push("assertj");
  if (/hamcrest/.test(text))
    dependencies.push("hamcrest");
  if (/surefire/.test(text))
    dependencies.push("maven-surefire-plugin");
  return { framework, dependencies };
}
function detectTestRoots(root) {
  const roots = [];
  if (fileExists(root, "src/test/java"))
    roots.push("src/test/java");
  if (fileExists(root, "src/test/kotlin"))
    roots.push("src/test/kotlin");
  if (fileExists(root, "src/test/groovy"))
    roots.push("src/test/groovy");
  return roots;
}
function detectWrapper(root) {
  return fileExists(root, "mvnw") || fileExists(root, "gradlew");
}
function detectExistingTestPattern(root, testRoots) {
  for (const tr of testRoots) {
    let entries = [];
    try {
      entries = listDir(root + "/" + tr);
    } catch {
      continue;
    }
    if (entries.some((e) => e.endsWith("Test.java")))
      return "Test.java";
    if (entries.some((e) => e.endsWith("Tests.java")))
      return "Tests.java";
    if (entries.some((e) => e.endsWith("Test.kt")))
      return "Test.kt";
  }
  return "Test.java";
}
var analyzeTestProjectTool = {
  name: "analyze_test_project",
  description: "Analyze project structure to determine build system, test directories, and framework.",
  parameters: { type: "object", properties: {}, required: [] },
  async execute(_params, ctx) {
    const started = Date.now();
    try {
      const buildSystem = detectBuildSystem(ctx.projectRoot);
      if (buildSystem === "unknown") {
        ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: false, errorCategory: "NOT_SUPPORTED" });
        return err(notSupported("cannot determine build system (no pom.xml / build.gradle)"));
      }
      const { framework, dependencies } = detectTestFramework(ctx.projectRoot, buildSystem);
      const testRoots = detectTestRoots(ctx.projectRoot);
      const sourceRoots = DEFAULT_SOURCE_ROOTS.filter((s) => fileExists(ctx.projectRoot, s));
      const info = {
        buildSystem,
        testFramework: framework,
        testRoots,
        sourceRoots,
        wrapperAvailable: detectWrapper(ctx.projectRoot),
        dependencies,
        existingTestPattern: detectExistingTestPattern(ctx.projectRoot, testRoots)
      };
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: true });
      return ok(info);
    } catch (e) {
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: false, errorCategory: toToolError(e, "analyze_test_project failed").category });
      return err(toToolError(e, "analyze_test_project failed"));
    }
  }
};
// src/tools/write-test-file.ts
var DEFAULT_TEST_ROOTS = ["src/test/java"];
var SCHEMA2 = {
  type: "object",
  properties: {
    path: { type: "string", minLength: 1 },
    content: { type: "string" },
    overwrite: { type: "boolean" },
    testRoots: { type: "array", items: { type: "string" } }
  },
  required: ["path", "content"],
  additionalProperties: false
};
var writeTestFileTool = {
  name: "write_test_file",
  description: "Write a test file. Path MUST be within one of the detected test roots.",
  parameters: SCHEMA2,
  async execute(params, ctx) {
    const started = Date.now();
    try {
      const issues = validateSchema(SCHEMA2, params);
      if (issues.length > 0) {
        throw invalidInput(`invalid input: ${issues.map((i) => `${i.path} ${i.message}`).join("; ")}`);
      }
      const input = params;
      const testRoots = input.testRoots && input.testRoots.length > 0 ? input.testRoots : DEFAULT_TEST_ROOTS;
      const result = writeFileAtomic({
        projectRoot: ctx.projectRoot,
        relPath: input.path,
        content: input.content,
        allowedRoots: testRoots,
        overwrite: input.overwrite === true
      });
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "write", projectRoot: ctx.projectRoot, targetPath: input.path, durationMs: Date.now() - started, ok: true });
      return ok(result);
    } catch (e) {
      const toolErr = toToolError(e, "write_test_file failed");
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "write", projectRoot: ctx.projectRoot, targetPath: params?.path, durationMs: Date.now() - started, ok: false, errorCategory: toolErr.category });
      return err(toolErr);
    }
  }
};
// src/tools/run-project-test.ts
var DEFAULT_TIMEOUT_SECONDS = 120;
var MAX_TIMEOUT_SECONDS = 600;
var SCHEMA3 = {
  type: "object",
  properties: {
    buildSystem: { type: "string", enum: ["maven", "gradle"] },
    module: { type: "string" },
    testClass: { type: "string" },
    testMethod: { type: "string" },
    profiles: { type: "array", items: { type: "string" } },
    extraArgs: { type: "array", items: { type: "string" } },
    timeoutSeconds: { type: "integer", minimum: 1 }
  },
  required: ["buildSystem"],
  additionalProperties: false
};
var EXTRA_ARG_FORBIDDEN = /[;&|`$<>\\\n]|^(rm|rmdir|sudo|doas|curl|wget|sh|bash|zsh|cmd|powershell|pwsh|nc|netcat|telnet)\b/i;
function buildCommand(input, root) {
  const isMaven = input.buildSystem === "maven";
  const wrapper = isMaven ? "mvnw" : "gradlew";
  const bare = isMaven ? "mvn" : "gradle";
  const hasWrapper = fileExists(root, wrapper);
  const base = hasWrapper ? `./${wrapper}` : bare;
  const argv = [base];
  if (isMaven) {
    if (input.profiles && input.profiles.length > 0)
      argv.push(`-P${input.profiles.join(",")}`);
    if (input.module)
      argv.push("-pl", input.module);
    const testSelector = input.testMethod ? `${input.testClass}#${input.testMethod}` : input.testClass;
    if (testSelector)
      argv.push(`-Dtest=${testSelector}`);
    argv.push("test");
  } else {
    if (input.module)
      argv.push(`${input.module}:test`);
    else
      argv.push("test");
    if (input.testClass)
      argv.push("--tests", input.testMethod ? `${input.testClass}.${input.testMethod}` : input.testClass);
  }
  if (input.extraArgs)
    argv.push(...input.extraArgs);
  return argv;
}
function parseTestSummary(stdout) {
  let passed = 0;
  let failed = 0;
  let errors = 0;
  let skipped = 0;
  const surefire = /Tests run:\s*(\d+),\s*Failures:\s*(\d+),\s*Errors:\s*(\d+),\s*Skipped:\s*(\d+)/.exec(stdout);
  if (surefire) {
    const total = parseInt(surefire[1] ?? "0", 10);
    failed = parseInt(surefire[2] ?? "0", 10);
    errors = parseInt(surefire[3] ?? "0", 10);
    skipped = parseInt(surefire[4] ?? "0", 10);
    passed = total - failed - errors - skipped;
  } else {
    const gradle = /(\d+)\s+tests?\s+completed,\s+(\d+)\s+failed/.exec(stdout);
    if (gradle) {
      const total = parseInt(gradle[1] ?? "0", 10);
      failed = parseInt(gradle[2] ?? "0", 10);
      passed = total - failed;
    }
  }
  const failureDetails = stdout.split(`
`).filter((l) => /\[ERROR\]|FAILED|<<< FAILURE|AssertionError|BUILD FAILURE/.test(l)).slice(0, 20);
  return { passed, failed, errors, skipped, failureDetails };
}
var runProjectTestTool = {
  name: "run_project_test",
  description: "Run project tests using Maven or Gradle wrapper.",
  parameters: SCHEMA3,
  async execute(params, ctx) {
    const started = Date.now();
    try {
      const issues = validateSchema(SCHEMA3, params);
      if (issues.length > 0) {
        throw invalidInput(`invalid input: ${issues.map((i) => `${i.path} ${i.message}`).join("; ")}`);
      }
      const input = params;
      if (input.extraArgs) {
        for (const arg of input.extraArgs) {
          if (EXTRA_ARG_FORBIDDEN.test(arg)) {
            throw permissionDenied(`extraArg rejected by whitelist: ${arg}`);
          }
        }
      }
      const timeoutMs = Math.min(input.timeoutSeconds ?? DEFAULT_TIMEOUT_SECONDS, MAX_TIMEOUT_SECONDS) * 1000;
      const argv = buildCommand(input, ctx.projectRoot);
      const result = await execCommand(argv, { cwd: ctx.projectRoot, timeoutMs });
      const summary = parseTestSummary(result.stdout);
      let category;
      if (result.timedOut)
        category = "TIMEOUT";
      else if (result.exitCode === 0)
        category = summary.failed > 0 || summary.errors > 0 ? "FAIL" : "PASS";
      else
        category = "FAIL";
      const output = {
        passed: summary.passed,
        failed: summary.failed,
        errors: summary.errors,
        skipped: summary.skipped,
        duration: (Date.now() - started) / 1000,
        failureDetails: summary.failureDetails,
        exitCode: result.exitCode ?? 1,
        category
      };
      const okResult = category === "PASS";
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "execute", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: okResult, errorCategory: okResult ? undefined : "COMMAND_FAILED" });
      return ok(output);
    } catch (e) {
      const toolErr = toToolError(e, "run_project_test failed");
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "execute", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: false, errorCategory: toolErr.category });
      return err(toolErr);
    }
  }
};
// src/tools/extract-api-spec.ts
import * as fs4 from "fs";
import * as path4 from "path";
var NOT_DETERMINED = "Not determined from code";
var SCHEMA4 = {
  type: "object",
  properties: {
    controllerFile: { type: "string", minLength: 1 }
  },
  required: ["controllerFile"],
  additionalProperties: false
};
var HTTP_METHOD_ANNOTATIONS = [
  ["GET", /@GetMapping/],
  ["POST", /@PostMapping/],
  ["PUT", /@PutMapping/],
  ["DELETE", /@DeleteMapping/],
  ["PATCH", /@PatchMapping/]
];
function extractStringArg(annotation) {
  const m = /(?:value|path)\s*=\s*"([^"]+)"/.exec(annotation) || /@\w+Mapping\s*\(\s*"([^"]+)"/.exec(annotation) || /@\w+Mapping\s*\(\s*"([^"]+)"/.exec(annotation);
  return m?.[1];
}
function extractValidation(block) {
  const out = [];
  if (/@NotNull\b/.test(block))
    out.push("@NotNull");
  if (/@NotBlank\b/.test(block))
    out.push("@NotBlank");
  if (/@NotEmpty\b/.test(block))
    out.push("@NotEmpty");
  if (/@Email\b/.test(block))
    out.push("@Email");
  const size = /@Size\([^)]*\)/.exec(block);
  if (size)
    out.push(size[0]);
  const min = /@Min\([^)]*\)/.exec(block);
  if (min)
    out.push(min[0]);
  const max = /@Max\([^)]*\)/.exec(block);
  if (max)
    out.push(max[0]);
  return out;
}
function parseClassInfo(source) {
  const classMatch = /(?:public\s+)?(?:final\s+)?class\s+(\w+)/.exec(source);
  const controllerName = classMatch?.[1] ?? NOT_DETERMINED;
  const rm = /@RequestMapping\s*\(([^)]*)\)/.exec(source);
  let basePath = "";
  if (rm) {
    basePath = extractStringArg(rm[0]) ?? "";
    if (!basePath) {
      const v = /(?:value|path)\s*=\s*\{?\s*"([^"]+)"/.exec(rm[1] ?? "");
      basePath = v?.[1] ?? "";
    }
  }
  return { controllerName, basePath };
}
function parseEndpoints(source) {
  const endpoints = [];
  const annRe = /(@(?:Get|Post|Put|Delete|Patch|Request)Mapping)(?:\(([^)]*)\))?/g;
  let m;
  while ((m = annRe.exec(source)) !== null) {
    const annName = m[1] ?? "";
    const annArgs = m[2] ?? "";
    const rest = source.slice(annRe.lastIndex);
    const sigMatch = /(?:public|private|protected)?\s*([\w<>\[\].,\s]+?)\s+(\w+)\s*\(([\s\S]*?)\)\s*(?:throws[^{]+)?\{/.exec(rest);
    if (!sigMatch)
      continue;
    const returnType = (sigMatch[1] ?? "").trim();
    const paramsText = sigMatch[3] ?? "";
    let method = "GET";
    let path5 = "";
    if (annName === "@RequestMapping") {
      const mm = /method\s*=\s*RequestMethod\.(\w+)/.exec(annArgs);
      if (!mm)
        continue;
      method = (mm[1] ?? "").toUpperCase();
      path5 = extractStringArg(annName + (annArgs ? `(${annArgs})` : "")) ?? "";
    } else {
      for (const [hm, re] of HTTP_METHOD_ANNOTATIONS) {
        if (re.test(annName)) {
          method = hm;
          break;
        }
      }
      path5 = extractStringArg(annName + (annArgs ? `(${annArgs})` : "")) ?? "";
    }
    const parameters = parseParameters(paramsText);
    const bodyParam = parameters.find((p) => p.location === "body");
    const requestBody = bodyParam ? { type: bodyParam.type, fields: [] } : undefined;
    const errorCodes = collectErrorCodes(rest);
    endpoints.push({
      method,
      path: path5,
      summary: "",
      parameters,
      requestBody,
      responseType: returnType,
      errorCodes
    });
  }
  return endpoints;
}
function parseParameters(paramsText) {
  if (!paramsText.trim())
    return [];
  const parameters = [];
  const parts = paramsText.split(",");
  for (const part of parts) {
    const p = part.trim();
    if (!p)
      continue;
    let location = "query";
    if (/@PathVariable/.test(p))
      location = "path";
    else if (/@RequestBody/.test(p))
      location = "body";
    else if (/@RequestHeader/.test(p))
      location = "header";
    else if (/@RequestParam/.test(p))
      location = "query";
    const nameMatch = /(?:@\w+(?:\([^)]*\))?\s*)*[\w<>\[\],\s]+\s+(\w+)\s*$/.exec(p) || /(?:@\w+(?:\([^)]*\))?\s*)*(\w+)\s*$/.exec(p);
    const name = nameMatch?.[1] ?? NOT_DETERMINED;
    const typeMatch = /(?:@\w+(?:\([^)]*\))?\s*)+([\w<>\[\],\s]+?)\s+\w+\s*$/.exec(p) || /([\w<>\[\],]+)\s+\w+\s*$/.exec(p);
    const type = typeMatch?.[1]?.trim() ?? NOT_DETERMINED;
    const required = location === "path" || location === "body" && !/@\w+\(required\s*=\s*false\)/.test(p) || !/@\w+\(required\s*=\s*false\)/.test(p) && location === "query" && /@RequestParam/.test(p) && !/required\s*=\s*false/.test(p);
    parameters.push({
      name,
      type,
      required,
      location,
      validation: extractValidation(p),
      description: ""
    });
  }
  return parameters;
}
function collectErrorCodes(methodBody) {
  const codes = [];
  const declaredRe = /@ExceptionHandler\s*\(\s*(\w+)\.class\s*\)/g;
  let m;
  while ((m = declaredRe.exec(methodBody)) !== null) {
    codes.push({ code: m[1] ?? "", status: "", source: "DECLARED" });
  }
  const statusRe = /@ResponseStatus\s*\(\s*(?:code\s*=\s*)?(?:HttpStatus\.)?(\w+)\s*\)/g;
  while ((m = statusRe.exec(methodBody)) !== null) {
    codes.push({ code: m[1] ?? "", status: m[1] ?? "", source: "DECLARED" });
  }
  const throwRe = /throw\s+new\s+(\w+Exception)\s*\(/g;
  while ((m = throwRe.exec(methodBody)) !== null) {
    codes.push({ code: m[1] ?? "", status: "", source: "REFERENCED" });
  }
  if (codes.length === 0) {
    codes.push({ code: "500", status: "INTERNAL_SERVER_ERROR", source: "INFERRED" });
  }
  const seen = new Set;
  return codes.filter((c) => {
    const k = `${c.code}:${c.source}`;
    if (seen.has(k))
      return false;
    seen.add(k);
    return true;
  });
}
function parseFieldDeclarations(source) {
  const fields = [];
  const re = /(@(?:NotNull|NotBlank|NotEmpty|Email|Size|Min|Max|Pattern)(?:\([^)]*\))?\s*)*\s*(?:private|protected|public)\s+([\w<>\[\],\s]+?)\s+(\w+)\s*;/g;
  let m;
  while ((m = re.exec(source)) !== null) {
    const validationBlock = m[0];
    fields.push({
      name: m[3] ?? "",
      type: (m[2] ?? "").trim(),
      validation: extractValidation(validationBlock),
      description: ""
    });
  }
  return fields;
}
function parseEnumValues(source) {
  const bodyMatch = /enum\s+\w+\s*\{([\s\S]*?)\}/.exec(source);
  if (!bodyMatch)
    return [];
  return (bodyMatch[1] ?? "").split(",").map((s) => s.trim()).filter((s) => s.length > 0 && !s.startsWith("//") && !s.startsWith("/*")).map((s) => s.replace(/\(.*?\)/, "").trim()).filter((s) => /^[A-Za-z_][A-Za-z0-9_]*$/.test(s));
}
function extractImports(source) {
  const out = [];
  const re = /^import\s+([\w.]+);\s*$/gm;
  let m;
  while ((m = re.exec(source)) !== null) {
    const full = m[1] ?? "";
    const className = full.split(".").pop() ?? "";
    const packagePath = full.slice(0, full.length - className.length - 1);
    out.push({ className, packagePath });
  }
  return out;
}
function findJavaFile(root, className) {
  const candidates = ["src/main/java", "src/test/java", "src/main/kotlin"];
  for (const base of candidates) {
    const found = findInDir(path4.join(root, base), className);
    if (found)
      return found;
  }
  return null;
}
function findInDir(dir, className) {
  if (!fs4.existsSync(dir))
    return null;
  const direct = path4.join(dir, `${className}.java`);
  if (fs4.existsSync(direct))
    return direct;
  for (const entry of fs4.readdirSync(dir, { withFileTypes: true })) {
    if (entry.isDirectory()) {
      const found = findInDir(path4.join(dir, entry.name), className);
      if (found)
        return found;
    }
  }
  return null;
}
var extractApiSpecTool = {
  name: "extract_api_spec",
  description: "Extract API specification from Spring MVC controller. Deterministic \u2014 never fabricates.",
  parameters: SCHEMA4,
  async execute(params, ctx) {
    const started = Date.now();
    try {
      const issues = validateSchema(SCHEMA4, params);
      if (issues.length > 0) {
        throw invalidInput(`invalid input: ${issues.map((i) => `${i.path} ${i.message}`).join("; ")}`);
      }
      const input = params;
      const abs = resolveInRoot(ctx.projectRoot, input.controllerFile);
      const source = fs4.readFileSync(abs, "utf8");
      const { controllerName, basePath } = parseClassInfo(source);
      const endpoints = parseEndpoints(source);
      const imports = extractImports(source);
      const dtos = {};
      const enums = {};
      for (const imp of imports) {
        const file = findJavaFile(ctx.projectRoot, imp.className);
        if (!file)
          continue;
        const rel = path4.relative(ctx.projectRoot, file).replace(/\\/g, "/");
        const src = fs4.readFileSync(file, "utf8");
        if (/enum\s+\w+/.test(src)) {
          enums[imp.className] = { values: parseEnumValues(src) };
        } else {
          dtos[imp.className] = { fields: parseFieldDeclarations(src) };
        }
      }
      for (const ep of endpoints) {
        if (ep.requestBody) {
          const dto = dtos[ep.requestBody.type];
          if (dto)
            ep.requestBody.fields = dto.fields;
        }
      }
      const output = { controllerName, basePath, endpoints, dtos, enums };
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, targetPath: input.controllerFile, durationMs: Date.now() - started, ok: true });
      return ok(output);
    } catch (e) {
      const toolErr = toToolError(e, "extract_api_spec failed");
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, targetPath: params?.controllerFile, durationMs: Date.now() - started, ok: false, errorCategory: toolErr.category });
      return err(toolErr);
    }
  }
};
// src/tools/validate-api-example.ts
var SCHEMA5 = {
  type: "object",
  properties: {
    example: {},
    spec: { type: "object" },
    endpointIndex: { type: "integer", minimum: 0 }
  },
  required: ["example", "spec", "endpointIndex"]
};
function fieldTypeToJson(field) {
  const t = field.type.toLowerCase();
  if (/long|integer|int\b/.test(t))
    return "number";
  if (/double|float|decimal|bigdecimal/.test(t))
    return "number";
  if (/boolean/.test(t))
    return "boolean";
  if (/list|set|array|collection|<\s*>/.test(t) || /<.+>/.test(t))
    return "array";
  return "string";
}
function isEnumType(spec, type) {
  return Object.prototype.hasOwnProperty.call(spec.enums, type);
}
function validateExample(input) {
  const { example, spec, endpointIndex } = input;
  const errors = [];
  const warnings = [];
  const endpoint = spec.endpoints[endpointIndex];
  if (!endpoint) {
    return { valid: false, errors: [`endpointIndex ${endpointIndex} out of range`], warnings };
  }
  if (!endpoint.requestBody) {
    return { valid: true, errors, warnings: ["endpoint has no request body to validate against"] };
  }
  const dto = spec.dtos[endpoint.requestBody.type];
  if (!dto) {
    return { valid: false, errors: [`request body type ${endpoint.requestBody.type} has no extracted DTO`], warnings };
  }
  const exampleObj = example;
  if (typeof exampleObj !== "object" || exampleObj === null || Array.isArray(exampleObj)) {
    return { valid: false, errors: ["example must be an object"], warnings };
  }
  const fieldByName = new Map(dto.fields.map((f) => [f.name, f]));
  for (const key of Object.keys(exampleObj)) {
    if (!fieldByName.has(key)) {
      warnings.push(`unknown field "${key}" (not in extracted DTO)`);
    }
  }
  for (const field of dto.fields) {
    const value = exampleObj[field.name];
    const required = field.validation.some((v) => v.startsWith("@NotNull") || v.startsWith("@NotBlank") || v.startsWith("@NotEmpty"));
    if (required && (value === undefined || value === null || value === "")) {
      errors.push(`missing required field "${field.name}"`);
      continue;
    }
    if (value === undefined)
      continue;
    if (isEnumType(spec, field.type)) {
      const allowed = spec.enums[field.type]?.values ?? [];
      if (!allowed.includes(String(value))) {
        errors.push(`field "${field.name}" value "${String(value)}" not in enum ${field.type}`);
      }
      continue;
    }
    const jsonType = fieldTypeToJson(field);
    const actualType = Array.isArray(value) ? "array" : typeof value;
    if (jsonType === "number" && actualType !== "number") {
      errors.push(`field "${field.name}" expected number, got ${actualType}`);
    }
    if (jsonType === "boolean" && actualType !== "boolean") {
      errors.push(`field "${field.name}" expected boolean, got ${actualType}`);
    }
    if (typeof value === "number") {
      const min = /@Min\((\d+)\)/.exec(field.validation.join(" "));
      if (min && value < parseInt(min[1] ?? "0", 10)) {
        errors.push(`field "${field.name}" below @Min(${min[1]})`);
      }
      const max = /@Max\((\d+)\)/.exec(field.validation.join(" "));
      if (max && value > parseInt(max[1] ?? "0", 10)) {
        errors.push(`field "${field.name}" above @Max(${max[1]})`);
      }
    }
  }
  return { valid: errors.length === 0, errors, warnings };
}
var validateApiExampleTool = {
  name: "validate_api_example",
  description: "Validate that generated API examples match the extracted spec schema.",
  parameters: SCHEMA5,
  async execute(params, ctx) {
    const started = Date.now();
    try {
      const issues = validateSchema(SCHEMA5, params);
      if (issues.length > 0) {
        throw invalidInput(`invalid input: ${issues.map((i) => `${i.path} ${i.message}`).join("; ")}`);
      }
      const input = params;
      const result = validateExample(input);
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: true });
      return ok(result);
    } catch (e) {
      const toolErr = toToolError(e, "validate_api_example failed");
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "read", projectRoot: ctx.projectRoot, durationMs: Date.now() - started, ok: false, errorCategory: toolErr.category });
      return err(toolErr);
    }
  }
};
// src/tools/write-document.ts
var DEFAULT_DOCS_ROOTS = ["docs", "doc", "api-docs"];
var SCHEMA6 = {
  type: "object",
  properties: {
    path: { type: "string", minLength: 1 },
    content: { type: "string" },
    docsRoot: { type: "string" }
  },
  required: ["path", "content"],
  additionalProperties: false
};
var writeDocumentTool = {
  name: "write_document",
  description: "Write documentation file. Path restricted to docs/, doc/, api-docs/.",
  parameters: SCHEMA6,
  async execute(params, ctx) {
    const started = Date.now();
    try {
      const issues = validateSchema(SCHEMA6, params);
      if (issues.length > 0) {
        throw invalidInput(`invalid input: ${issues.map((i) => `${i.path} ${i.message}`).join("; ")}`);
      }
      const input = params;
      const allowedRoots = input.docsRoot ? [input.docsRoot] : DEFAULT_DOCS_ROOTS;
      const result = writeFileAtomic({
        projectRoot: ctx.projectRoot,
        relPath: input.path,
        content: input.content,
        allowedRoots,
        overwrite: true
      });
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "write", projectRoot: ctx.projectRoot, targetPath: input.path, durationMs: Date.now() - started, ok: true });
      return ok(result);
    } catch (e) {
      const toolErr = toToolError(e, "write_document failed");
      ctx.guard.after({ sessionId: ctx.sessionId, agent: ctx.agent, tool: this.name, action: "write", projectRoot: ctx.projectRoot, targetPath: params?.path, durationMs: Date.now() - started, ok: false, errorCategory: toolErr.category });
      return err(toolErr);
    }
  }
};
export {
  writeTestFileTool,
  writeDocumentTool,
  validatePermissions,
  validateExample,
  validateApiExampleTool,
  toToolError,
  toRelativePath,
  timeoutError,
  scanDlp,
  runProjectTestTool,
  riskLabel,
  resolveProjectPath,
  redact,
  permissionDenied,
  pathViolation,
  parseUnifiedDiff,
  parseTestSummary,
  parseFieldDeclarations,
  parseFailed,
  parseEnumValues,
  parseEndpoints,
  parseClassInfo,
  ok,
  notSupported,
  isDangerous,
  invalidInput,
  internalError,
  hasBlockingSecret,
  extractImports,
  extractApiSpecTool,
  err,
  dlpBlocked,
  difyConfigFromEnv,
  commandFailed,
  collectReviewContextTool,
  classifyError,
  classifyDifyError,
  classifyCommandFailure,
  assertWithinRoot,
  assertWithinAllowedRoots,
  analyzeTestProjectTool,
  analyzeCommand,
  ToolError,
  RuntimeSecurityGuard,
  RiskSafe,
  RiskDeny,
  RiskAsk,
  PathViolationError,
  DifyInvalidResponseError,
  DifyHttpError,
  DifyClient,
  CircuitOpenError,
  CircuitBreaker,
  AuditLogger
};
