import * as fs from "node:fs";
import * as path from "node:path";

// Path boundary enforcement for all enterprise tools. A path is allowed only if,
// after separator normalisation, lexical resolution and symlink resolution, it
// still lands inside the project root. Windows drive/UNC absolutes are rejected
// regardless of the host platform so the policy is identical on macOS/Windows.

export class PathViolationError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "PathViolationError";
  }
}

const WINDOWS_DRIVE = /^[a-zA-Z]:[\\/]/;
const WINDOWS_UNC = /^\\\\/;

function normalizeSeparators(p: string): string {
  return p.replace(/\\/g, "/");
}

function isWithin(root: string, target: string): boolean {
  if (target === root) return true;
  const prefix = root.endsWith(path.sep) ? root : root + path.sep;
  return target.startsWith(prefix);
}

// Resolve symlinks on the deepest existing ancestor, re-appending any
// not-yet-existing trailing segments. Needed because tools may write new files
// whose parent chain is only partially materialised.
function realpathExisting(p: string): string {
  let cur = path.resolve(p);
  const suffix: string[] = [];
  while (!fs.existsSync(cur)) {
    const parent = path.dirname(cur);
    if (parent === cur) break;
    suffix.unshift(path.basename(cur));
    cur = parent;
  }
  let real = fs.realpathSync(cur);
  for (const seg of suffix) {
    real = path.join(real, seg);
  }
  return real;
}

export function resolveProjectPath(root: string, inputPath: string): string {
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

  let realRoot: string;
  let realTarget: string;
  try {
    realRoot = fs.realpathSync(rootAbs);
    realTarget = realpathExisting(target);
  } catch (err) {
    // Root must exist; propagate as a path violation rather than a raw fs error.
    throw new PathViolationError(`cannot resolve path: ${(err as Error).message}`);
  }
  if (!isWithin(realRoot, realTarget)) {
    throw new PathViolationError("symlink escape detected");
  }
  return target;
}

export function assertWithinRoot(root: string, p: string): void {
  resolveProjectPath(root, p);
}

export function assertWithinAllowedRoots(p: string, roots: readonly string[]): void {
  const target = path.resolve(p);
  for (const root of roots) {
    const rootAbs = path.resolve(root);
    if (!isWithin(rootAbs, target)) continue;
    try {
      const realRoot = realpathExisting(rootAbs);
      const realTarget = realpathExisting(target);
      if (isWithin(realRoot, realTarget)) return;
    } catch {
      // fall through to the next candidate root
    }
  }
  throw new PathViolationError("path not within any allowed root");
}

export function toRelativePath(root: string, p: string): string {
  const rootAbs = path.resolve(root);
  const target = path.resolve(rootAbs, normalizeSeparators(p));
  const rel = path.relative(rootAbs, target);
  return normalizeSeparators(rel === "" ? "." : rel);
}
