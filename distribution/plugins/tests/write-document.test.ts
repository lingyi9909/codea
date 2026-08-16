import { describe, expect, test } from "bun:test";
import * as fs from "node:fs";
import * as path from "node:path";
import { writeDocumentTool } from "../src/tools/write-document";
import { makeTempRoot, makeContext } from "./helpers";

function setup(): { root: string; ctx: ReturnType<typeof makeContext>["ctx"] } {
  const root = makeTempRoot("codea-doc-");
  fs.mkdirSync(path.join(root, "docs"), { recursive: true });
  fs.mkdirSync(path.join(root, "src"), { recursive: true });
  const { ctx } = makeContext(root);
  return { root, ctx };
}

describe("writeDocumentTool.execute", () => {
  test("writes a markdown doc under docs/", async () => {
    const { root, ctx } = setup();
    const result = await writeDocumentTool.execute({ path: "docs/api/user.md", content: "# User API\n" }, ctx);
    expect(result.ok).toBe(true);
    expect(fs.existsSync(path.join(root, "docs/api/user.md"))).toBe(true);
  });

  test("rejects writing into src/", async () => {
    const { ctx } = setup();
    const result = await writeDocumentTool.execute({ path: "src/main/Main.java", content: "class Main {}" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("PATH_VIOLATION");
  });

  test("rejects path traversal", async () => {
    const { ctx } = setup();
    const result = await writeDocumentTool.execute({ path: "../../outside.md", content: "x" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("PATH_VIOLATION");
  });

  test("rejects absolute path", async () => {
    const { ctx } = setup();
    const result = await writeDocumentTool.execute({ path: "/etc/cron.d/evil", content: "x" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("PATH_VIOLATION");
  });

  test("blocks DLP secret content", async () => {
    const { ctx } = setup();
    const result = await writeDocumentTool.execute(
      { path: "docs/leak.md", content: "Authorization: Bearer abcdefghijklmnopqrst" },
      ctx,
    );
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("DLP_BLOCKED");
  });

  test("rejects missing path (invalid schema)", async () => {
    const { ctx } = setup();
    const result = await writeDocumentTool.execute({ content: "x" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("INVALID_INPUT");
  });
});
