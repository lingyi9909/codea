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

// Split into command tokens on whitespace and shell metacharacters so dangerous
// commands are matched as whole words (and inside substitutions too).
function tokenize(input: string): string[] {
  return input
    .toLowerCase()
    .split(/[\s;&|()<>$`"'\\]+/)
    .filter((t) => t.length > 0);
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

export function isDangerous(input: string): boolean {
  return analyzeCommand(input).risk === RiskDeny;
}

export function riskLabel(risk: CommandRisk): string {
  return risk;
}
