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

describe("analyzeCommand — argument-level: sensitive paths -> deny", () => {
  const cases: [string, string][] = [
    ["cat .env", "sensitive-path:sensitive-file:.env"],
    ["cat /etc/passwd", "sensitive-path:absolute-path"],
    ["grep password ~/.aws/credentials", "sensitive-path:home-path"],
    ["find / -name '*.pem'", "sensitive-path:absolute-path"],
    ["head -n 5 ~/.ssh/id_rsa", "sensitive-path:home-path"],
    ["cat ../../etc/passwd", "sensitive-path:parent-traversal"],
    ["git diff -- .env", "sensitive-path:sensitive-file:.env"],
    ["tail ~/.git-credentials", "sensitive-path:home-path"],
  ];
  for (const [cmd, rule] of cases) {
    test(`${cmd} -> deny (${rule})`, () => {
      const a = analyzeCommand(cmd);
      expect(a.risk).toBe("deny");
      expect(a.matchedRule).toBe(rule);
    });
  }
});

describe("analyzeCommand — dynamic expansion -> ask (glob/variable bypass)", () => {
  const cases = [
    "cat .e?v",
    "cat .e*",
    "cat $SECRET_FILE",
    "cat ${SECRET_FILE}",
    "ls *.java",
    "grep foo *",
    "head [a-z]*.txt",
  ];
  for (const cmd of cases) {
    test(`${cmd} -> ask`, () => {
      const a = analyzeCommand(cmd);
      expect(a.risk).toBe("ask");
    });
  }
});

describe("analyzeCommand — argument-level: dangerous git options -> deny", () => {
  const cases: [string, string][] = [
    ["git -c core.pager=sh log", "git-option:-c/--config"],
    ["git --config core.sshCommand=evil status", "git-option:--config"],
    ["git --git-dir=/tmp/x log", "git-option:--git-dir"],
    ["git diff --output=/tmp/leak", "git-option:--output"],
    ["git --work-tree=/tmp status", "git-option:--work-tree"],
    ["git -C /tmp status", "git-option:-c/--config"],
  ];
  for (const [cmd, rule] of cases) {
    test(`${cmd} -> deny (${rule})`, () => {
      const a = analyzeCommand(cmd);
      expect(a.risk).toBe("deny");
      expect(a.matchedRule).toBe(rule);
    });
  }
});
