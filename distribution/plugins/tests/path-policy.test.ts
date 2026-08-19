import { afterAll, beforeAll, describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { PathViolationError, resolveProjectPath, assertWithinRoot, toRelativePath, validateNativeReadPath } from "../src/security/path-policy";

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

describe("validateNativeReadPath — absolute paths are not inherently dangerous", () => {
  test("absolute path inside project is allowed", () => {
    expect(validateNativeReadPath(root, path.join(root, "src/main/java/Foo.java"))).toBeNull();
  });
  test("absolute path at the project root is allowed", () => {
    expect(validateNativeReadPath(root, root)).toBeNull();
  });
  test("relative path inside project is allowed", () => {
    expect(validateNativeReadPath(root, "src/main/Foo.java")).toBeNull();
  });
  test("absolute path outside project is denied", () => {
    expect(validateNativeReadPath(root, outside)).toBe("outside-project");
  });
  test("absolute system path outside project is denied", () => {
    expect(validateNativeReadPath(root, "/etc/passwd")).toBe("outside-project");
  });
  test("relative traversal outside project is denied", () => {
    expect(validateNativeReadPath(root, "../../secret.txt")).toBe("outside-project");
  });
  test("empty path is denied", () => {
    expect(validateNativeReadPath(root, "")).toBe("empty-path");
  });
});

describe("validateNativeReadPath — sensitive targets inside the root", () => {
  test(".env inside project is denied", () => {
    expect(validateNativeReadPath(root, path.join(root, ".env"))).toBe("sensitive-file:.env");
  });
  test(".env.production inside project is denied", () => {
    expect(validateNativeReadPath(root, path.join(root, ".env.production"))).toBe("sensitive-file:.env");
  });
  test(".ssh directory inside project is denied", () => {
    expect(validateNativeReadPath(root, path.join(root, ".ssh", "config"))).toBe("sensitive-dir");
  });
  test("credentials file inside project is denied", () => {
    expect(validateNativeReadPath(root, path.join(root, "credentials"))).toBe("sensitive-file:credentials");
  });
  test("ssh private key inside project is denied", () => {
    expect(validateNativeReadPath(root, path.join(root, "id_rsa"))).toBe("sensitive-file:ssh-key");
  });
});

describe("validateNativeReadPath — symlink escape", () => {
  test("symlink inside root pointing outside is denied", () => {
    const link = path.join(root, "escape-link");
    fs.symlinkSync(outside, link);
    try {
      expect(validateNativeReadPath(root, path.join(link, "secret.txt"))).toBe("symlink-escape");
    } finally {
      fs.rmSync(link, { force: true });
    }
  });
});

describe("validateNativeReadPath — windows paths", () => {
  const winRoot = "C:\\code\\project";
  test("windows absolute inside project is allowed", () => {
    expect(validateNativeReadPath(winRoot, "C:\\code\\project\\src\\main\\Foo.java")).toBeNull();
  });
  test("windows forward-slash absolute inside project is allowed", () => {
    expect(validateNativeReadPath(winRoot, "C:/code/project/src/Foo.java")).toBeNull();
  });
  test("windows absolute outside project is denied", () => {
    expect(validateNativeReadPath(winRoot, "C:\\Windows\\System32\\config")).toBe("outside-project");
  });
  test("windows different drive is denied", () => {
    expect(validateNativeReadPath(winRoot, "D:\\secret\\file.txt")).toBe("outside-project");
  });
  test("windows UNC outside project is denied", () => {
    expect(validateNativeReadPath(winRoot, "\\\\server\\share\\file.txt")).toBe("outside-project");
  });
  test("windows sensitive file inside project is denied", () => {
    expect(validateNativeReadPath(winRoot, "C:\\code\\project\\.env")).toBe("sensitive-file:.env");
  });
  test("case-insensitive windows containment allows different-cased drive/segments", () => {
    expect(validateNativeReadPath(winRoot, "c:\\CODE\\Project\\src\\Foo.java")).toBeNull();
  });
});
