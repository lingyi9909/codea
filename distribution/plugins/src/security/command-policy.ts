import type { CommandAnalysis, CommandRisk } from "./types";
import { RiskAsk, RiskDeny, RiskSafe } from "./types";

// Shell command safety analysis. V1 is deliberately not a full shell parser, but
// it must never treat a "safe prefix" as making the whole command safe: the full
// command (including inside $(...) / backticks) is scanned for dangerous tokens
// before any whitelist is consulted.
//
// Policy:
//   - explicit dangerous token          -> deny
//   - shell composition (pipe/redir/sub)-> ask
//   - dangerous git option (on git ...) -> deny
//   - sensitive path arg (on read cmds) -> deny
//   - strict read-only whitelist        -> safe
//   - anything else                     -> ask

const DANGEROUS_COMMANDS: ReadonlySet<string> = new Set([
  // privilege escalation
  "sudo",
  "doas",
  "su",
  // network fetch / transfer (exfil or ingest)
  "curl",
  "wget",
  "nc",
  "netcat",
  "ncat",
  "telnet",
  // destructive
  "rm",
  "rmdir",
  "del",
  "erase",
  // nested shells / script hosts (arbitrary command execution)
  "sh",
  "bash",
  "zsh",
  "cmd",
  "powershell",
  "pwsh",
  // windows dangerous cmdlets
  "remove-item",
  "invoke-webrequest",
  "invoke-expression",
  "start-process",
]);

// Read-only commands with safe (non-composing) arguments. Anchored so the whole
// command must match; metacharacters are already routed to ask/deny above.
const SAFE_COMMANDS: readonly RegExp[] = [
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
  /^find(\s+.*)?$/,
];

// Git options that let a read-only command turn into arbitrary code execution,
// directory escape, or file writes. Blocked on any `git ...` command before the
// whitelist is consulted, so `git -c core.pager=sh log` can never read as safe.
const DANGEROUS_GIT_OPTIONS: readonly string[] = [
  "-c", // --config: sets core.pager/sshCommand/alias/filter (code exec)
  "--config",
  "--config-env",
  "-C", // --directory: change directory (escape root)
  "--directory",
  "--exec-path",
  "--git-dir",
  "--work-tree",
  "--output", // writes a file
  "--upload-pack",
  "--receive-pack",
  "--pager",
];

// Split into command tokens on whitespace and shell metacharacters so dangerous
// commands are matched as whole words (and inside substitutions too).
function tokenize(input: string): string[] {
  return input
    .toLowerCase()
    .split(/[\s;&|()<>$`"'\\]+/)
    .filter((t) => t.length > 0);
}

// Detects a dangerous git option among the tokens. Returns a short label or null.
function findDangerousGitOption(tokens: readonly string[]): string | null {
  for (const tok of tokens) {
    if (tok === "git") continue;
    if (tok === "-c" || tok.startsWith("-c=") || /^-c[a-z]/.test(tok)) return "-c/--config";
    if (tok === "--config" || tok.startsWith("--config=") || tok.startsWith("--config-env")) return "--config";
    if (tok === "--directory" || tok.startsWith("--directory=")) return "--directory";
    if (tok === "--exec-path" || tok.startsWith("--exec-path=")) return "--exec-path";
    if (tok === "--git-dir" || tok.startsWith("--git-dir=")) return "--git-dir";
    if (tok === "--work-tree" || tok.startsWith("--work-tree=")) return "--work-tree";
    if (tok === "--output" || tok.startsWith("--output=")) return "--output";
    if (tok === "--upload-pack" || tok.startsWith("--upload-pack=")) return "--upload-pack";
    if (tok === "--receive-pack" || tok.startsWith("--receive-pack=")) return "--receive-pack";
    if (tok === "--pager" || tok.startsWith("--pager=")) return "--pager";
  }
  return null;
}

// Detects dynamic shell expansion in an otherwise-safe command: globs, character
// classes and variable references can expand to a sensitive path at runtime
// (e.g. `cat .e*` -> `.env`, `cat $SECRET_FILE`) that static path scanning cannot
// see. Such commands are downgraded to ask rather than whitelisted safe.
function hasDynamicExpansion(command: string): boolean {
  if (/[\*\?\[]/.test(command)) return true;
  if (/\$\{?[A-Za-z_][A-Za-z0-9_]*/.test(command)) return true;
  return false;
}

// Detects sensitive-path arguments on a read-only command: absolute paths escape
// the project root, and dotfiles/credential/ssh-key paths are an exfil target.
// Exported for the plugin adapter, which applies the same check to native
// read/grep/glob tool paths (filePath / path) so an enterprise agent cannot
// `read .env` or `grep` a credential file through a native tool.
export function findSensitivePath(command: string): string | null {
  if (/(^|[\s'"])\//.test(command)) return "absolute-path";
  if (/(^|[\s'"])~([\\/]|$)/.test(command)) return "home-path";
  if (/(^|[\s'"])[a-zA-Z]:[\\/]/.test(command)) return "windows-absolute";
  if (/(^|[\\/\s'"])\.\.([\\/]|$)/.test(command)) return "parent-traversal";
  if (/\.env(\.[\w-]+)?([\\/]|$)/i.test(command)) return "sensitive-file:.env";
  if (/(^|[\\/\s'"])(\.ssh|\.aws|\.gnupg)([\\/]|$)/i.test(command)) return "sensitive-dir";
  if (/(^|[\\/\s'"])(id_rsa|id_ed25519|id_ecdsa|id_dsa)([\\/]|$)/i.test(command)) return "sensitive-file:ssh-key";
  if (/(^|[\\/\s'"])credentials([\\/]|$)|\.git-credentials|\.npmrc|\.netrc|\.pem([\\/]|$)/i.test(command)) return "sensitive-file:credentials";
  return null;
}

export function analyzeCommand(input: string): CommandAnalysis {
  const command = input.trim();
  const analysis: CommandAnalysis = {
    risk: RiskAsk,
    command,
    hasPipe: false,
    hasRedirect: false,
    hasSubCmd: false,
    hasChain: false,
    matchedRule: "",
  };

  analysis.hasPipe = /\|(?!\|)/.test(command);
  analysis.hasRedirect = />|>>|</.test(command);
  analysis.hasSubCmd = /\$\(/.test(command) || /`/.test(command);
  analysis.hasChain = /&&|\|\||;/.test(command);

  const tokens = tokenize(command);
  for (const token of tokens) {
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

  // Argument-level controls for git: block options that escalate a read-only
  // command into code execution / directory escape / file write.
  if (tokens[0] === "git") {
    const gitOption = findDangerousGitOption(tokens);
    if (gitOption) {
      analysis.risk = RiskDeny;
      analysis.matchedRule = `git-option:${gitOption}`;
      return analysis;
    }
  }

  const lower = command.toLowerCase();
  for (const re of SAFE_COMMANDS) {
    if (re.test(lower)) {
      // A whitelisted read-only command must not touch sensitive paths.
      const sensitivePath = findSensitivePath(command);
      if (sensitivePath) {
        analysis.risk = RiskDeny;
        analysis.matchedRule = `sensitive-path:${sensitivePath}`;
        return analysis;
      }
      if (hasDynamicExpansion(command)) {
        analysis.risk = RiskAsk;
        return analysis;
      }
      analysis.risk = RiskSafe;
      return analysis;
    }
  }

  analysis.risk = RiskAsk;
  return analysis;
}

export function isDangerous(input: string): boolean {
  return analyzeCommand(input).risk === RiskDeny;
}

export function riskLabel(risk: CommandRisk): string {
  return risk;
}
