import { describe, expect, test } from "bun:test";
import { scanDlp, redact } from "../src/security/dlp";

describe("scanDlp — blocking secrets", () => {
  const cases = [
    "Authorization: Bearer abcdefghijklmnopqrstuvwxyz123456",
    "Bearer abcdefghijklmnopqrstuvwxyz123456",
    "api_key=sk_live_1234567890",
    "apikey: secret123",
    "password=hunter2",
    "token=ghp_abcdefghijklmnopqrstuvwxyz0123456789",
    "secret=super-secret-value",
    "-----BEGIN RSA PRIVATE KEY-----",
    "ghp_abcdefghijklmnopqrstuvwxyz0123456789",
    "AKIAIOSFODNN7EXAMPLE",
  ];
  for (const input of cases) {
    test(`${input.slice(0, 32)}... -> block`, () => {
      const r = scanDlp(input, "tool-input");
      expect(r.allowed).toBe(false);
      expect(r.findings.length).toBeGreaterThan(0);
    });
  }
});

describe("scanDlp — redact sensitive paths (not block)", () => {
  const cases = [
    [".env", ".env"],
    ["src/main/resources/.env", ".env"],
    ["id_rsa", "id_rsa"],
    ["server.pem", "server.pem"],
    ["credentials", "credentials"],
  ];
  for (const [input, _hit] of cases) {
    test(`${input} -> redact`, () => {
      const r = scanDlp(input, "tool-input");
      expect(r.allowed).toBe(true);
      expect(r.redacted).not.toContain(_hit);
      expect(r.findings.length).toBeGreaterThan(0);
    });
  }
});

describe("scanDlp — safe content passes unchanged", () => {
  test("plain text", () => {
    const input = "public class Foo { public int bar() { return 1; } }";
    const r = scanDlp(input, "tool-output");
    expect(r.allowed).toBe(true);
    expect(r.redacted).toBe(input);
    expect(r.findings).toEqual([]);
  });
});

describe("scanDlp — redaction masks secret value", () => {
  test("password value replaced", () => {
    const r = scanDlp("db password=supersecret connect", "prompt");
    expect(r.allowed).toBe(false);
    expect(r.redacted).not.toContain("supersecret");
    expect(r.redacted).toContain("[REDACTED]");
  });
});

describe("redact helper", () => {
  test("returns redacted string", () => {
    expect(redact("token=abc", "audit")).not.toContain("abc");
  });
});
