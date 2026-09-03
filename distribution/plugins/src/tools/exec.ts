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

const BATCH_FILE_RE = /\.(cmd|bat)$/i;

function normalizeWindowsBatchPath(file: string): string {
  return file.startsWith("./") ? `.\\${file.slice(2)}` : file;
}

// On Windows, .cmd/.bat batch files cannot be spawned directly by execFile — they
// must run through cmd.exe. Route them via a controlled `cmd.exe /d /s /c`
// invocation (argv array, never shell:true) so no POSIX shell is introduced. The
// single command-line argument is joined by displayCommand; callers
// (run_project_test) reject shell/cmd metacharacters in their args before this
// point, so the /c command line carries no live metacharacters. `./wrapper.cmd`
// is normalized to the native cmd.exe current-directory form `.\\wrapper.cmd`
// at this boundary only; the higher-level fixed argv contract remains unchanged.
// On every other platform the argv passes through unchanged.
export function resolveExecArgv(argv: readonly string[], platform: string = process.platform): string[] {
  const file = argv[0] ?? "";
  if (platform === "win32" && BATCH_FILE_RE.test(file)) {
    const commandArgv = [normalizeWindowsBatchPath(file), ...argv.slice(1)];
    return ["cmd.exe", "/d", "/s", "/c", displayCommand(commandArgv)];
  }
  return [...argv];
}

export function execCommand(argv: readonly string[], opts: ExecOptions): Promise<ExecResult> {
  const actual = resolveExecArgv(argv);
  const file = actual[0];
  if (!file) throw commandFailed("empty command argv");
  const args = actual.slice(1);
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
