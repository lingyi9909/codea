import { describe, expect, test } from "bun:test";
import { execCommand } from "../src/tools/exec";
import { parseUnifiedDiff, collectReviewContextTool } from "../src/tools/collect-review-context";
import { makeTempRoot, makeContext } from "./helpers";
import * as fs from "node:fs";
import * as path from "node:path";

const DIFF = [
  "diff --git a/src/main/java/A.java b/src/main/java/A.java",
  "index 1111111..2222222 100644",
  "--- a/src/main/java/A.java",
  "+++ b/src/main/java/A.java",
  "@@ -1,3 +1,4 @@",
  " context",
  "-old line",
  "+new line",
  "+added line",
  "diff --git a/new/B.java b/new/B.java",
  "new file mode 100644",
  "index 0000000..3333333",
  "--- /dev/null",
  "+++ b/new/B.java",
  "@@ -0,0 +1,2 @@",
  "+package new;",
  "+public class B {}",
  "diff --git a/old/C.java b/renamed/C.java",
  "similarity index 90%",
  "rename from old/C.java",
  "rename to renamed/C.java",
].join("\n");

describe("parseUnifiedDiff", () => {
  test("parses modified/added/renamed files with correct counts", () => {
    const out = parseUnifiedDiff(DIFF);
    expect(out.filesChanged).toBe(3);
    expect(out.linesAdded).toBe(4); // +new line, +added line, +package, +public
    expect(out.linesRemoved).toBe(1); // -old line

    const modified = out.files.find((f) => f.path === "src/main/java/A.java");
    expect(modified?.status).toBe("modified");
    expect(modified?.hunks[0].oldStart).toBe(1);
    expect(modified?.hunks[0].newStart).toBe(1);

    const added = out.files.find((f) => f.path === "new/B.java");
    expect(added?.status).toBe("added");

    const renamed = out.files.find((f) => f.path === "renamed/C.java");
    expect(renamed?.status).toBe("renamed");
    expect(renamed?.oldPath).toBe("old/C.java");
  });
});

describe("collectReviewContextTool.execute", () => {
  test("collects unstaged diff from a real git repo", async () => {
    const root = makeTempRoot("codea-review-");
    const { ctx } = makeContext(root);

    await execCommand(["git", "init", "-q"], { cwd: root });
    await execCommand(["git", "config", "user.email", "t@example.com"], { cwd: root });
    await execCommand(["git", "config", "user.name", "t"], { cwd: root });

    fs.writeFileSync(path.join(root, "README.md"), "hello\n");
    await execCommand(["git", "add", "."], { cwd: root });
    await execCommand(["git", "commit", "-qm", "init"], { cwd: root });

    fs.writeFileSync(path.join(root, "README.md"), "hello\nworld\n");
    const result = await collectReviewContextTool.execute({ source: "unstaged" }, ctx);

    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data.filesChanged).toBe(1);
      expect(result.data.linesAdded).toBe(1);
      expect(result.data.files[0].path).toBe("README.md");
    }
  });

  test("rejects an out-of-root file path", async () => {
    const root = makeTempRoot("codea-review-");
    const { ctx } = makeContext(root);
    const result = await collectReviewContextTool.execute({ source: "file-path", filePath: "../outside.txt" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("PATH_VIOLATION");
  });

  test("rejects missing commit for source=commit", async () => {
    const root = makeTempRoot("codea-review-");
    const { ctx } = makeContext(root);
    const result = await collectReviewContextTool.execute({ source: "commit" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("INVALID_INPUT");
  });

  test("rejects unknown source via schema", async () => {
    const root = makeTempRoot("codea-review-");
    const { ctx } = makeContext(root);
    const result = await collectReviewContextTool.execute({ source: "bogus" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("INVALID_INPUT");
  });

  test("rejects option injection in baseBranch", async () => {
    const root = makeTempRoot("codea-review-");
    const { ctx } = makeContext(root);
    const result = await collectReviewContextTool.execute({ source: "base-branch", baseBranch: "--output=/tmp/leak" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("INVALID_INPUT");
  });

  test("rejects option injection in commit", async () => {
    const root = makeTempRoot("codea-review-");
    const { ctx } = makeContext(root);
    const result = await collectReviewContextTool.execute({ source: "commit", commit: "--upload-pack=evil" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("INVALID_INPUT");
  });

  test("rejects option injection in range refs", async () => {
    const root = makeTempRoot("codea-review-");
    const { ctx } = makeContext(root);
    const result = await collectReviewContextTool.execute({ source: "range", rangeFrom: "--git-dir=/tmp/x", rangeTo: "HEAD" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("INVALID_INPUT");
  });

  test("rejects shell metacharacters in commit ref", async () => {
    const root = makeTempRoot("codea-review-");
    const { ctx } = makeContext(root);
    const result = await collectReviewContextTool.execute({ source: "commit", commit: "main; rm -rf /" }, ctx);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.category).toBe("INVALID_INPUT");
  });
});
