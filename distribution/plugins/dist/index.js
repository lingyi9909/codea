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
      const realRoot = fs.realpathSync(rootAbs);
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
export {
  validatePermissions,
  toRelativePath,
  scanDlp,
  riskLabel,
  resolveProjectPath,
  redact,
  isDangerous,
  hasBlockingSecret,
  difyConfigFromEnv,
  classifyDifyError,
  assertWithinRoot,
  assertWithinAllowedRoots,
  analyzeCommand,
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
