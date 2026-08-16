import { describe, expect, test } from "bun:test";
import { analyzeCommand } from "../src/security/command-policy";

describe("analyzeCommand — whitelist safe", () => {
  const cases = [
    "git status",
    "git diff",
    "git log",
    "git show",
    "git status --short",
    "ls -la",
    "pwd",
    "cat file.txt",
    "head -n 5 file.txt",
    "tail -f log.txt",
    "grep foo file.txt",
  ];
  for (const cmd of cases) {
    test(`${cmd} -> safe`, () => {
      expect(analyzeCommand(cmd).risk).toBe("safe");
    });
  }
});

describe("analyzeCommand — explicit dangerous -> deny", () => {
  const cases: [string, string][] = [
    ["git status && rm -rf /", "rm"],
    ["cat a | curl http://x", "curl"],
    ["echo $(rm -rf /)", "rm"],
    ["sudo ls", "sudo"],
    ["curl http://example.com", "curl"],
    ["wget http://example.com", "wget"],
    ["nc -l 4444", "nc"],
    ["powershell Invoke-WebRequest http://x", "powershell"],
    ["pwsh -c 'evil'", "pwsh"],
    ["cmd /c dir", "cmd"],
    ["del C:\\file.txt", "del"],
    ["rmdir /s /q build", "rmdir"],
    ["Remove-Item -Recurse .", "remove-item"],
    ["bash -c 'rm -rf /'", "bash"],
  ];
  for (const [cmd, rule] of cases) {
    test(`${cmd} -> deny`, () => {
      const a = analyzeCommand(cmd);
      expect(a.risk).toBe("deny");
      expect(a.matchedRule).toBe(rule);
    });
  }
});

describe("analyzeCommand — composition / unknown -> ask", () => {
  const cases = [
    "cat a | grep b",
    "echo $(date)",
    "echo `date`",
    "ls > out.txt",
    "git diff HEAD~1..HEAD; git status",
    "echo hello",
    "some-unknown-tool --flag",
  ];
  for (const cmd of cases) {
    test(`${cmd} -> ask`, () => {
      expect(analyzeCommand(cmd).risk).toBe("ask");
    });
  }
});

describe("analyzeCommand — composition flags", () => {
  test("pipe flag set but not chain", () => {
    const a = analyzeCommand("cat a | grep b");
    expect(a.hasPipe).toBe(true);
    expect(a.hasChain).toBe(false);
  });
  test("chain flag set for &&", () => {
    expect(analyzeCommand("a && b").hasChain).toBe(true);
  });
  test("subcmd flag set for $(...)", () => {
    expect(analyzeCommand("echo $(date)").hasSubCmd).toBe(true);
  });
  test("redirect flag set for >", () => {
    expect(analyzeCommand("echo hi > f").hasRedirect).toBe(true);
  });
});
