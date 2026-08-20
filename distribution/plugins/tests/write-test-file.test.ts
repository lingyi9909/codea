import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as path from "node:path";
import { writeTestFileTool } from "../src/tools/write-test-file";
import type { ToolContext, WriteOwnership } from "../src/tools/types";
import { makeTempRoot, makeContext } from "./helpers";

// makeOwnership builds an isolated in-memory ownership set, mirroring the
// per-(session, agent) store the plugin entry wires in at serve time.
function makeOwnership(): WriteOwnership {
  const set = new Set<string>();
  return {
    record: (p: string) => { set.add(p); },
    owns: (p: string) => set.has(p),
  };
}

function ownedContext(projectRoot: string, ownership: WriteOwnership, agent = "unit-test-generator"): ToolContext {
  const { ctx } = makeContext(projectRoot);
  ctx.ownership = ownership;
  ctx.agent = agent;
  return ctx;
}

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

  test("rejects caller-supplied testRoots (not caller-controllable)", async () => {
    const { ctx } = setup();
    const result = await writeTestFileTool.execute(
      { path: "src/main/java/Evil.java", content: "x", testRoots: ["src/main/java"] },
      ctx,
    );
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("INVALID_INPUT");
  });

  test("rejects when no test roots are detected", async () => {
    const root = makeTempRoot("codea-write-noroot-");
    const { ctx } = makeContext(root);
    const result = await writeTestFileTool.execute(
      { path: "src/test/java/X.java", content: "x" },
      ctx,
    );
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("NOT_SUPPORTED");
  });
});

describe("writeTestFileTool.execute — server-side ownership", () => {
  test("first write (overwrite=false) creates the file and records ownership", async () => {
    const { root } = setup();
    const ownership = makeOwnership();
    const ctx = ownedContext(root, ownership);
    const rel = "src/test/java/com/example/FreshTest.java";
    const result = await writeTestFileTool.execute({ path: rel, content: "package com.example;" }, ctx);
    expect(result.ok).toBe(true);
    const abs = path.join(root, rel);
    expect(fs.existsSync(abs)).toBe(true);
    expect(ownership.owns(abs)).toBe(true);
  });

  test("overwrite=true on a file this run created is allowed (repair)", async () => {
    const { root } = setup();
    const ownership = makeOwnership();
    const ctx = ownedContext(root, ownership);
    const rel = "src/test/java/com/example/RepairTest.java";
    await writeTestFileTool.execute({ path: rel, content: "v1" }, ctx);
    const result = await writeTestFileTool.execute({ path: rel, content: "v2", overwrite: true }, ctx);
    expect(result.ok).toBe(true);
    expect(fs.readFileSync(path.join(root, rel), "utf8")).toBe("v2");
  });

  test("overwrite=true on a pre-existing test is denied (not owned)", async () => {
    const { root } = setup();
    const rel = "src/test/java/ExistingTest.java";
    fs.writeFileSync(path.join(root, rel), "original");
    const ctx = ownedContext(root, makeOwnership());
    const result = await writeTestFileTool.execute({ path: rel, content: "new", overwrite: true }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("PERMISSION_DENIED");
    expect(fs.readFileSync(path.join(root, rel), "utf8")).toBe("original");
  });

  test("overwrite=true from another session/agent is denied (different ownership)", async () => {
    const { root } = setup();
    const rel = "src/test/java/com/example/CrossSession.java";
    const creator = ownedContext(root, makeOwnership(), "unit-test-generator");
    await writeTestFileTool.execute({ path: rel, content: "v1" }, creator);
    const intruder = ownedContext(root, makeOwnership(), "unit-test-generator");
    const result = await writeTestFileTool.execute({ path: rel, content: "v2", overwrite: true }, intruder);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("PERMISSION_DENIED");
    expect(fs.readFileSync(path.join(root, rel), "utf8")).toBe("v1");
  });

  test("overwrite=false on a file this run already created is denied", async () => {
    const { root } = setup();
    const ownership = makeOwnership();
    const ctx = ownedContext(root, ownership);
    const rel = "src/test/java/com/example/DupCreate.java";
    await writeTestFileTool.execute({ path: rel, content: "v1" }, ctx);
    const result = await writeTestFileTool.execute({ path: rel, content: "v2" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("PERMISSION_DENIED");
    expect(fs.readFileSync(path.join(root, rel), "utf8")).toBe("v1");
  });
});
