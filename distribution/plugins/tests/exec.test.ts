import { describe, expect, test } from "bun:test";
import { resolveExecArgv } from "../src/tools/exec";

describe("resolveExecArgv — Windows batch wrapper routing", () => {
  test("routes .cmd wrappers through cmd.exe /d /s /c on win32", () => {
    const argv = resolveExecArgv(["./mvnw.cmd", "-Dtest=FooTest", "test"], "win32");
    expect(argv).toEqual(["cmd.exe", "/d", "/s", "/c", "./mvnw.cmd -Dtest=FooTest test"]);
  });

  test("routes .bat wrappers through cmd.exe /d /s /c on win32", () => {
    const argv = resolveExecArgv(["./gradlew.bat", "test"], "win32");
    expect(argv).toEqual(["cmd.exe", "/d", "/s", "/c", "./gradlew.bat test"]);
  });

  test("leaves non-batch argv unchanged on win32", () => {
    const argv = resolveExecArgv(["mvn", "test"], "win32");
    expect(argv).toEqual(["mvn", "test"]);
  });

  test("leaves .cmd argv unchanged on darwin (execFile)", () => {
    const argv = resolveExecArgv(["./mvnw.cmd", "test"], "darwin");
    expect(argv).toEqual(["./mvnw.cmd", "test"]);
  });
});
