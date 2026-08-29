import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";

const verificationModule = await import("../src/tools/verify-project").catch(() => ({} as Record<string, unknown>));

type Profile = {
  kind: "maven" | "gradle" | "go" | "unknown";
  executable: string | null;
  stages: Array<{ name: string; argv: string[] }>;
  reason?: string;
};
type Detect = (root: string, platform?: string) => Profile;

function detect(): Detect {
  expect(typeof verificationModule.detectVerificationProfile).toBe("function");
  return verificationModule.detectVerificationProfile as Detect;
}

function fixture(name: string): string {
  return path.join(import.meta.dir, "fixtures", "verify", name);
}

function copyFixture(name: string): string {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), `codea-verify-${name}-`));
  fs.cpSync(fixture(name), root, { recursive: true });
  return root;
}

describe("verify_project local profile detection", () => {
  test("prefers Maven wrapper and fixed Maven lifecycle stages", () => {
    const profile = detect()(fixture("maven"), "linux");
    expect(profile.kind).toBe("maven");
    expect(profile.executable).toBe("./mvnw");
    expect(profile.stages.map((stage) => stage.name)).toEqual(["compile", "test", "verify"]);
    expect(profile.stages[0]?.argv).toEqual(["./mvnw", "-DskipTests", "compile"]);
    expect(profile.stages[1]?.argv).toEqual(["./mvnw", "test"]);
    expect(profile.stages[2]?.argv).toEqual(["./mvnw", "verify"]);
  });

  test("uses Windows Maven cmd wrapper and falls back to bare mvn without wrappers", () => {
    expect(detect()(fixture("maven"), "win32").executable).toBe("./mvnw.cmd");
    const root = copyFixture("maven");
    try {
      fs.rmSync(path.join(root, "mvnw"), { force: true });
      fs.rmSync(path.join(root, "mvnw.cmd"), { force: true });
      expect(detect()(root, "win32").executable).toBe("mvn");
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });

  test("prefers Gradle wrapper and adds check only when local check plugins are configured", () => {
    const linux = detect()(fixture("gradle"), "linux");
    expect(linux.kind).toBe("gradle");
    expect(linux.executable).toBe("./gradlew");
    expect(linux.stages.map((stage) => stage.name)).toEqual(["classes", "test", "check"]);
    expect(linux.stages[2]?.argv).toEqual(["./gradlew", "check"]);
    expect(detect()(fixture("gradle"), "win32").executable).toBe("./gradlew.bat");
  });

  test("falls back to bare gradle when wrappers are absent", () => {
    const root = copyFixture("gradle");
    try {
      fs.rmSync(path.join(root, "gradlew"), { force: true });
      fs.rmSync(path.join(root, "gradlew.bat"), { force: true });
      expect(detect()(root, "linux").executable).toBe("gradle");
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });

  test("detects Go with fixed test and vet stages", () => {
    const profile = detect()(fixture("go"), "linux");
    expect(profile.kind).toBe("go");
    expect(profile.executable).toBe("go");
    expect(profile.stages.map((stage) => stage.argv)).toEqual([
      ["go", "test", "./..."],
      ["go", "vet", "./..."],
    ]);
  });

  test("returns unknown for no build file and for Maven/Gradle ambiguity", () => {
    const unknown = detect()(fixture("unknown"), "linux");
    expect(unknown.kind).toBe("unknown");
    expect(unknown.executable).toBeNull();
    expect(unknown.stages).toEqual([]);

    const mixed = detect()(fixture("mixed"), "linux");
    expect(mixed.kind).toBe("unknown");
    expect(mixed.reason).toBe("AMBIGUOUS_BUILD_SYSTEM");
    expect(mixed.stages).toEqual([]);
  });
});
