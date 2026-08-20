import * as fs from "node:fs";
import * as path from "node:path";
import { scanDlp } from "../security/dlp";
import { PathViolationError, assertWithinAllowedRoots, resolveProjectPath } from "../security/path-policy";
import { dlpBlocked, internalError, pathViolation, permissionDenied } from "./errors";
import type { WriteOwnership } from "./types";

// Filesystem operations shared by the write tools. Every path is resolved
// through the canonicalising path policy first; writes are DLP-gated and atomic
// (tmp file + rename) so a failure never leaves a half-written file behind.

export function resolveInRoot(projectRoot: string, relPath: string): string {
  try {
    return resolveProjectPath(projectRoot, relPath);
  } catch (e) {
    if (e instanceof PathViolationError) throw pathViolation(e.message);
    throw e;
  }
}

export function fileExists(projectRoot: string, relPath: string): boolean {
  return fs.existsSync(resolveInRoot(projectRoot, relPath));
}

export function readTextFile(projectRoot: string, relPath: string): string {
  const abs = resolveInRoot(projectRoot, relPath);
  try {
    return fs.readFileSync(abs, "utf8");
  } catch (e) {
    throw internalError(`read failed: ${relPath}`, e);
  }
}

export function listDir(dir: string): string[] {
  try {
    return fs.readdirSync(dir);
  } catch (e) {
    throw internalError(`readdir failed: ${dir}`, e);
  }
}

export function ensureDir(dir: string): void {
  fs.mkdirSync(dir, { recursive: true });
}

export interface WriteOptions {
  projectRoot: string;
  relPath: string;
  content: string;
  allowedRoots?: string[]; // additional roots (relative to projectRoot) the path must fall inside
  overwrite: boolean;
  ownership?: WriteOwnership; // when set, overwrite is restricted to paths this run created
}

export interface WriteResult {
  path: string;
  bytes: number;
}

export function writeFileAtomic(opts: WriteOptions): WriteResult {
  const abs = resolveInRoot(opts.projectRoot, opts.relPath);

  if (opts.allowedRoots && opts.allowedRoots.length > 0) {
    const absRoots = opts.allowedRoots.map((r) => path.resolve(opts.projectRoot, r));
    try {
      assertWithinAllowedRoots(abs, absRoots);
    } catch (e) {
      if (e instanceof PathViolationError) throw pathViolation(e.message);
      throw e;
    }
  }

  if (fs.existsSync(abs)) {
    if (opts.ownership) {
      // Server-side ownership: an existing file may only be overwritten when this
      // exact (session, agent) run created it AND overwrite=true. Any other
      // existing file (a pre-existing test, or one created by another session) is
      // denied regardless of the overwrite flag.
      if (!opts.ownership.owns(abs)) {
        throw permissionDenied(`file exists and is not owned by this run: ${opts.relPath}`);
      }
      if (!opts.overwrite) {
        throw permissionDenied(`file exists and overwrite is not allowed: ${opts.relPath}`);
      }
    } else if (!opts.overwrite) {
      throw permissionDenied(`file exists and overwrite is not allowed: ${opts.relPath}`);
    }
  }

  const dlp = scanDlp(opts.content, "file-write");
  if (!dlp.allowed) {
    const rule = dlp.findings[0]?.rule ?? "secret";
    throw dlpBlocked(`content blocked by DLP: ${rule}`);
  }

  const dir = path.dirname(abs);
  fs.mkdirSync(dir, { recursive: true });
  const tmp = path.join(dir, `.codea-tmp-${process.pid}-${Date.now()}-${Math.random().toString(36).slice(2)}`);
  try {
    fs.writeFileSync(tmp, opts.content, "utf8");
    fs.renameSync(tmp, abs);
  } catch (e) {
    try {
      fs.rmSync(tmp, { force: true });
    } catch {
      // best-effort cleanup; the write error below is what matters
    }
    throw internalError(`write failed: ${opts.relPath}`, e);
  }

  opts.ownership?.record(abs);

  return { path: abs, bytes: Buffer.byteLength(opts.content, "utf8") };
}
