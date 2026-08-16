import type { DlpContext, DlpFinding, DlpResult, DlpSeverity } from "./types";

// Four-layer DLP:
//   Layer 1: secrets / credentials  (block)
//   Layer 2: sensitive file paths   (redact)
//   Layer 3: content redaction      (redact, applied to every scan)
//   Layer 4: output/audit minimisation (handled by audit-log.ts)
//
// High-risk credential matches block the action; ordinary sensitive values are
// redacted so secrets never propagate into logs or tool output.

interface DlpRule {
  name: string;
  severity: DlpSeverity;
  action: "block" | "redact";
  pattern: RegExp;
  mask: string;
}

const MASK_SECRET = "[REDACTED]";
const MASK_PATH = "[REDACTED-PATH]";

const DLP_RULES: readonly DlpRule[] = [
  // ---- Layer 1: secrets / credentials (block) ----
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

  // ---- Layer 2: sensitive file paths (redact) ----
  { name: "env-file", severity: "medium", action: "redact", mask: MASK_PATH, pattern: /\.env(?:\.\w+)?\b/g },
  { name: "ssh-key", severity: "medium", action: "redact", mask: MASK_PATH, pattern: /\bid_(?:rsa|ed25519|ecdsa|dsa)\b/g },
  { name: "pem-file", severity: "medium", action: "redact", mask: MASK_PATH, pattern: /[\w./-]+\.pem\b/g },
  { name: "credentials-file", severity: "medium", action: "redact", mask: MASK_PATH, pattern: /\bcredentials(?:\.\w+)?\b/gi },
];

function maskFor(rule: DlpRule): string {
  return rule.mask;
}

export function scanDlp(input: string, _context: DlpContext): DlpResult {
  if (typeof input !== "string" || input.length === 0) {
    return { allowed: true, redacted: input, findings: [] };
  }

  const findings: DlpFinding[] = [];
  let redacted = input;
  let allowed = true;

  for (const rule of DLP_RULES) {
    const matches = [...input.matchAll(rule.pattern)];
    if (matches.length === 0) continue;
    for (const m of matches) {
      findings.push({
        rule: rule.name,
        severity: rule.severity,
        start: m.index,
        end: m.index + m[0].length,
      });
    }
    if (rule.action === "block") {
      allowed = false;
    }
    redacted = redacted.replace(rule.pattern, maskFor(rule));
  }

  return { allowed, redacted, findings };
}

export function redact(input: string, context: DlpContext): string {
  return scanDlp(input, context).redacted;
}

export function hasBlockingSecret(input: string, context: DlpContext): boolean {
  return !scanDlp(input, context).allowed;
}
