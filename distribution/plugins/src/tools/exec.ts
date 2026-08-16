import { execFile } from "node:child_process";
import { timeoutError, commandFailed } from "./errors";
import { classifyCommandFailure } from "./failure-classifier";

// Command execution for enterprise tools. Always argv arrays (never shell
// string concatenation), always bounded by an explicit timeout and output size
// cap. There is no implicit shell — metacharacters in an argument stay literal.

export interface ExecOptions {
  cwd: string;
  timeoutMs?: number;
  env?: Record<string, string>;
  maxBuffer?: number;
}

export interface ExecResult {
  exitCode: number | null;
  stdout: string;
  stderr: string;
  timedOut: boolean;
  command: string;
}

const DEFAULT_TIMEOUT_MS = 30_000;
const DEFAULT_MAX_BUFFER = 10 * 1024 * 1024; // 10MB — bounded, no OOM on huge diff

export function displayCommand(argv: readonly string[]): string {
  return argv.map((a) => (/\s/.test(a) ? JSON.stringify(a) : a)).join(" ");
}

export function execCommand(argv: readonly string[], opts: ExecOptions): Promise<ExecResult> {
  const file = argv[0];
  if (!file) throw commandFailed("empty command argv");
  const args = argv.slice(1);
  const timeoutMs = opts.timeoutMs ?? DEFAULT_TIMEOUT_MS;

  return new Promise<ExecResult>((resolve) => {
    execFile(
      file,
      args,
      {
        cwd: opts.cwd,
        timeout: timeoutMs,
        maxBuffer: opts.maxBuffer ?? DEFAULT_MAX_BUFFER,
        encoding: "utf8",
        env: { ...process.env, ...(opts.env ?? {}) },
      },
      (error, stdout, stderr) => {
        const out = String(stdout ?? "");
        const errOut = String(stderr ?? "");
        const timedOut = !!(error && (error as { killed?: boolean }).killed);
        const exitCode = error ? (error.code ?? 1) : 0;
        resolve({
          exitCode: typeof exitCode === "number" ? exitCode : 1,
          stdout: out,
          stderr: errOut,
          timedOut,
          command: displayCommand(argv),
        });
      },
    );
  });
}

export interface ExecErrorContext {
  argv: readonly string[];
  result: ExecResult;
}

// Throws a classified ToolError for a nonzero/timeout command result.
export function ensureCommandSucceeded(result: ExecResult, argv: readonly string[]): void {
  if (result.exitCode === 0 && !result.timedOut) return;
  const category = classifyCommandFailure(result.timedOut ? null : result.exitCode, result.timedOut);
  const message = result.timedOut
    ? `command timed out: ${result.command}`
    : `command failed (exit ${result.exitCode}): ${result.command}`;
  if (result.timedOut) throw timeoutError(message);
  throw commandFailed(`${message}${result.stderr ? ` — ${result.stderr.trim().slice(0, 500)}` : ""}`, { argv, result });
}
