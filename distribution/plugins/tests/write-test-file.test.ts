import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as path from "node:path";
import { writeTestFileTool } from "../src/tools/write-test-file";
import { makeTempRoot, makeContext } from "./helpers";

function setup(): { root: string; ctx: ReturnType<typeof makeContext>["ctx"] } {
  const root = makeTempRoot("codea-write-");
  fs.mkdirSync(path.join(root, "src/test/java"), { recursive: true });
  fs.mkdirSync(path.join(root, "src/main/java"), { recursive: true });
  const { ctx } = makeContext(root);
  return { root, ctx };
}

describe("writeTestFileTool.execute", () => {
  test("writes a test file inside a test root", async () => {
    const { root, ctx } = setup();
    const result = await writeTestFileTool.execute(
      { path: "src/test/java/com/example/FooTest.java", content: "package com.example;\n" },
      ctx,
    );
    expect(result.ok).toBe(true);
    expect(fs.existsSync(path.join(root, "src/test/java/com/example/FooTest.java"))).toBe(true);
  });

  test("rejects a write into the production source root", async () => {
    const { ctx } = setup();
    const result = await writeTestFileTool.execute(
      { path: "src/main/java/com/example/Evil.java", content: "package com.example;" },
      ctx,
    );
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("PATH_VIOLATION");
  });

  test("rejects path traversal out of the project", async () => {
    const { ctx } = setup();
    const result = await writeTestFileTool.execute(
      { path: "../../outside/EvilTest.java", content: "x" },
      ctx,
    );
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("PATH_VIOLATION");
  });

  test("rejects an absolute path", async () => {
    const { ctx } = setup();
    const result = await writeTestFileTool.execute(
      { path: "/etc/passwd", content: "x" },
      ctx,
    );
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("PATH_VIOLATION");
  });

  test("rejects a Windows absolute path", async () => {
    const { ctx } = setup();
    const result = await writeTestFileTool.execute(
      { path: "C:\\Windows\\System32\\evil.java", content: "x" },
      ctx,
    );
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("PATH_VIOLATION");
  });

  test("denies overwrite of an existing file by default", async () => {
    const { root, ctx } = setup();
    const rel = "src/test/java/ExistingTest.java";
    fs.writeFileSync(path.join(root, rel), "original");
    const result = await writeTestFileTool.execute({ path: rel, content: "new" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("PERMISSION_DENIED");
    expect(fs.readFileSync(path.join(root, rel), "utf8")).toBe("original");
  });

  test("allows overwrite when overwrite=true", async () => {
    const { root, ctx } = setup();
    const rel = "src/test/java/ExistingTest.java";
    fs.writeFileSync(path.join(root, rel), "original");
    const result = await writeTestFileTool.execute({ path: rel, content: "new", overwrite: true }, ctx);
    expect(result.ok).toBe(true);
    expect(fs.readFileSync(path.join(root, rel), "utf8")).toBe("new");
  });

  test("blocks DLP secret content", async () => {
    const { ctx } = setup();
    const result = await writeTestFileTool.execute(
      { path: "src/test/java/LeakTest.java", content: 'String token = "password=hunter2";' },
      ctx,
    );
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("DLP_BLOCKED");
  });

  test("rejects symlink escape", async () => {
    const { root, ctx } = setup();
    const outside = path.join(root, "..", "outside-dir");
    fs.mkdirSync(outside, { recursive: true });
    fs.symlinkSync(outside, path.join(root, "src/test/java/link"), "dir");
    const result = await writeTestFileTool.execute(
      { path: "src/test/java/link/evil.java", content: "x" },
      ctx,
    );
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("PATH_VIOLATION");
  });

  test("rejects invalid schema (missing content)", async () => {
    const { ctx } = setup();
    const result = await writeTestFileTool.execute({ path: "src/test/java/X.java" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("INVALID_INPUT");
  });
});
