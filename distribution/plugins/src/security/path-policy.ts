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

// Separator-agnostic containment: both inputs are already separator-normalised,
// so prefix-match against a single "/" is correct on every host platform.
function isWithinNormalized(root: string, target: string): boolean {
  if (target === root) return true;
  const prefix = root.endsWith("/") ? root : root + "/";
  return target.startsWith(prefix);
}

const SENSITIVE_DIRS = new Set([".ssh", ".aws", ".gnupg"]);
const SENSITIVE_FILES = new Set(["credentials", ".git-credentials", ".npmrc", ".netrc"]);
const SSH_KEY_FILES = new Set(["id_rsa", "id_ed25519", "id_ecdsa", "id_dsa"]);
const ENV_FILE_RE = /^\.env(\.[\w-]+)?$/i;
const PEM_FILE_RE = /\.pem$/i;

// Detects a sensitive file/dir target anywhere in a (already separator-normalised)
// path. Applied only after containment is confirmed, so this never broadens the
// boundary — it only denies sensitive destinations that sit inside the root.
function sensitiveSegment(targetPath: string): string | null {
  const norm = normalizeSeparators(targetPath);
  const segments = norm.split("/").filter((s) => s.length > 0);
  if (segments.length === 0) return null;
  const base = segments[segments.length - 1];
  if (ENV_FILE_RE.test(base)) return "sensitive-file:.env";
  if (PEM_FILE_RE.test(base)) return "sensitive-file:.pem";
  if (SSH_KEY_FILES.has(base)) return "sensitive-file:ssh-key";
  if (SENSITIVE_FILES.has(base)) return "sensitive-file:credentials";
  for (const seg of segments) {
    if (SENSITIVE_DIRS.has(seg.toLowerCase())) return "sensitive-dir";
  }
  return null;
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

// Validates a path that a native OpenCode tool (read/grep/glob) is about to touch.
// Unlike findSensitivePath, an absolute path is NOT dangerous by itself: OpenCode
// defines read.filePath as an absolute path, so /project/src/Foo.java is a normal,
// in-root read. The policy is instead:
//
//   original path -> canonicalize/realpath -> still inside projectRoot?
//     |-- no  -> deny (outside-project / symlink-escape / unresolvable)
//     `-- yes -> check .env/.ssh/.aws/credentials/etc -> allow or deny
//
// Windows drive/UNC absolutes are resolved lexically with path.win32 (they cannot
// be realpaths on a POSIX host, and vice versa) so the containment rule stays
// identical on macOS and Windows. Returns a short deny reason, or null to allow.
export function validateNativeReadPath(root: string, targetPath: string): string | null {
  if (typeof targetPath !== "string" || targetPath.length === 0) {
    return "empty-path";
  }

  const windows = WINDOWS_DRIVE.test(targetPath) || WINDOWS_UNC.test(targetPath);
  const pathImpl = windows ? path.win32 : path.posix;
  const caseFold = (p: string): string => (windows ? p.toLowerCase() : p);

  const rootResolved = pathImpl.resolve(normalizeSeparators(root));
  const targetResolved = pathImpl.isAbsolute(targetPath)
    ? pathImpl.resolve(normalizeSeparators(targetPath))
    : pathImpl.resolve(rootResolved, normalizeSeparators(targetPath));

  const rootKey = caseFold(normalizeSeparators(rootResolved));
  const targetKey = caseFold(normalizeSeparators(targetResolved));
  if (!isWithinNormalized(rootKey, targetKey)) {
    return "outside-project";
  }

  const sensitive = sensitiveSegment(targetPath);
  if (sensitive) return sensitive;

  // Symlink escape is only meaningful on host-resolvable (POSIX) paths. Windows
  // absolutes are judged lexically above and cannot be realpath'd on a POSIX host.
  if (!windows) {
    try {
      const realRoot = fs.realpathSync(rootResolved);
      const realTarget = realpathExisting(targetResolved);
      if (!isWithinNormalized(normalizeSeparators(realRoot), normalizeSeparators(realTarget))) {
        return "symlink-escape";
      }
    } catch {
      return "unresolvable";
    }
  }

  return null;
}
