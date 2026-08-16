// Shared security types for the Codea V1 enterprise plugin.

export type CommandRisk = "safe" | "ask" | "deny";

export const RiskSafe: CommandRisk = "safe";
export const RiskAsk: CommandRisk = "ask";
export const RiskDeny: CommandRisk = "deny";

export interface CommandAnalysis {
  risk: CommandRisk;
  command: string;
  hasPipe: boolean;
  hasRedirect: boolean;
  hasSubCmd: boolean;
  hasChain: boolean;
  matchedRule: string;
}

export type DlpContext = "prompt" | "tool-input" | "tool-output" | "file-write" | "audit";

export type DlpSeverity = "low" | "medium" | "high";

export interface DlpFinding {
  rule: string;
  severity: DlpSeverity;
  start?: number;
  end?: number;
}

export interface DlpResult {
  allowed: boolean;
  redacted: string;
  findings: DlpFinding[];
}
