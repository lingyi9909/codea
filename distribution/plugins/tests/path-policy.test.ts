import { afterAll, beforeAll, describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { PathViolationError, resolveProjectPath, assertWithinRoot, toRelativePath } from "../src/security/path-policy";

let tmp: string;
let root: string;
let outside: string;

beforeAll(() => {
  tmp = fs.mkdtempSync(path.join(os.tmpdir(), "codea-path-"));
  root = path.join(tmp, "project");
  outside = path.join(tmp, "outside");
  fs.mkdirSync(path.join(root, "src", "main", "java"), { recursive: true });
  fs.mkdirSync(outside, { recursive: true });
  fs.writeFileSync(path.join(outside, "secret.txt"), "secret");
});

afterAll(() => {
  fs.rmSync(tmp, { recursive: true, force: true });
});

describe("resolveProjectPath — in-root paths", () => {
  test("relative nested path resolves within root", () => {
    const p = resolveProjectPath(root, "src/main/java/Foo.java");
    expect(p).toBe(path.join(root, "src/main/java/Foo.java"));
  });
  test("windows-style relative separators normalise", () => {
    const p = resolveProjectPath(root, "src\\main\\java\\Foo.java");
    expect(p).toBe(path.join(root, "src/main/java/Foo.java"));
  });
  test("non-existent nested path is allowed", () => {
    const p = resolveProjectPath(root, "newdir/subdir/newfile.txt");
    expect(p).toBe(path.join(root, "newdir/subdir/newfile.txt"));
  });
});

describe("resolveProjectPath — traversal escape", () => {
  test("../ escapes", () => {
    expect(() => resolveProjectPath(root, "../secret.txt")).toThrow(PathViolationError);
  });
  test("../../ escapes", () => {
    expect(() => resolveProjectPath(root, "../../etc/passwd")).toThrow(PathViolationError);
  });
  test("absolute posix path escapes", () => {
    expect(() => resolveProjectPath(root, "/etc/passwd")).toThrow(PathViolationError);
  });
  test("windows-style relative traversal escapes", () => {
    expect(() => resolveProjectPath(root, "..\\..\\secret.txt")).toThrow(PathViolationError);
  });
});

describe("resolveProjectPath — windows absolute/UNC", () => {
  test("drive letter absolute rejected", () => {
    expect(() => resolveProjectPath(root, "C:\\Windows\\System32\\config")).toThrow(PathViolationError);
  });
  test("UNC path rejected", () => {
    expect(() => resolveProjectPath(root, "\\\\server\\share\\file.txt")).toThrow(PathViolationError);
  });
});

describe("resolveProjectPath — symlink escape", () => {
  test("symlink pointing outside is rejected", () => {
    const link = path.join(root, "link");
    fs.symlinkSync(outside, link);
    try {
      expect(() => resolveProjectPath(root, "link/secret.txt")).toThrow(PathViolationError);
    } finally {
      fs.rmSync(link, { force: true });
    }
  });
});

describe("assertWithinRoot", () => {
  test("valid path passes", () => {
    expect(() => assertWithinRoot(root, "src/main/java/Foo.java")).not.toThrow();
  });
  test("escape path throws", () => {
    expect(() => assertWithinRoot(root, "../../x")).toThrow(PathViolationError);
  });
});

describe("toRelativePath", () => {
  test("returns project-relative path", () => {
    expect(toRelativePath(root, path.join(root, "src/main/java/Foo.java"))).toBe("src/main/java/Foo.java");
  });
});
