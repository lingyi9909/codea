import { afterAll, beforeAll, describe, expect, test } from "bun:test";
import {
  CircuitBreaker,
  CircuitOpenError,
  DifyClient,
  type DifyConfig,
} from "../src/dify-query";

describe("CircuitBreaker — closed -> open -> half-open -> closed", () => {
  test("opens after threshold consecutive failures", async () => {
    let clock = 0;
    const b = new CircuitBreaker({ threshold: 3, resetTimeoutMs: 60_000, now: () => clock });
    expect(b.currentState).toBe("closed");
    for (let i = 0; i < 3; i++) {
      await expect(b.execute(() => Promise.reject(new Error("boom")))).rejects.toThrow("boom");
    }
    expect(b.currentState).toBe("open");
  });

  test("open circuit rejects immediately without calling fn", async () => {
    let clock = 0;
    const b = new CircuitBreaker({ threshold: 3, resetTimeoutMs: 60_000, now: () => clock });
    for (let i = 0; i < 3; i++) {
      await expect(b.execute(() => Promise.reject(new Error("x")))).rejects.toThrow();
    }
    let called = false;
    await expect(b.execute(() => { called = true; return Promise.resolve(1); })).rejects.toThrow(CircuitOpenError);
    expect(called).toBe(false);
  });

  test("half-open probe success closes circuit", async () => {
    let clock = 0;
    const b = new CircuitBreaker({ threshold: 3, resetTimeoutMs: 60_000, now: () => clock });
    for (let i = 0; i < 3; i++) {
      await expect(b.execute(() => Promise.reject(new Error("x")))).rejects.toThrow();
    }
    clock = 61_000;
    expect(b.currentState).toBe("half-open");
    await expect(b.execute(() => Promise.resolve("ok"))).resolves.toBe("ok");
    expect(b.currentState).toBe("closed");
  });

  test("half-open probe failure re-opens", async () => {
    let clock = 0;
    const b = new CircuitBreaker({ threshold: 3, resetTimeoutMs: 60_000, now: () => clock });
    for (let i = 0; i < 3; i++) {
      await expect(b.execute(() => Promise.reject(new Error("x")))).rejects.toThrow();
    }
    clock = 61_000;
    await expect(b.execute(() => Promise.reject(new Error("probe")))).rejects.toThrow("probe");
    expect(b.currentState).toBe("open");
  });
});

describe("DifyClient — fake HTTP server", () => {
  let mode: "ok" | "400" | "500" | "invalid" = "ok";
  let hits = 0;
  const server = Bun.serve({
    port: 0,
    fetch() {
      hits += 1;
      if (mode === "400") return new Response("bad request", { status: 400 });
      if (mode === "500") return new Response("server error", { status: 500 });
      if (mode === "invalid") return new Response("not json", { status: 200 });
      return new Response(JSON.stringify({ answer: "hello" }), { status: 200 });
    },
  });
  const baseUrl = `http://127.0.0.1:${server.port}`;
  const config: DifyConfig = { baseUrl, apiKey: "test-key" };

  afterAll(() => {
    server.stop(true);
  });

  test("success returns answer", async () => {
    mode = "ok";
    const client = new DifyClient(config);
    const r = await client.query("what is 1+1");
    expect(r.degraded).toBe(false);
    expect(r.answer).toBe("hello");
  });

  test("4xx classifies http-4xx", async () => {
    mode = "400";
    const client = new DifyClient(config);
    const r = await client.query("x");
    expect(r.degraded).toBe(true);
    expect(r.error).toBe("http-4xx");
  });

  test("5xx classifies http-5xx", async () => {
    mode = "500";
    const client = new DifyClient(config);
    const r = await client.query("x");
    expect(r.degraded).toBe(true);
    expect(r.error).toBe("http-5xx");
  });

  test("invalid JSON classifies invalid-response", async () => {
    mode = "invalid";
    const client = new DifyClient(config);
    const r = await client.query("x");
    expect(r.degraded).toBe(true);
    expect(r.error).toBe("invalid-response");
  });

  test("connection refused classifies network-error", async () => {
    const client = new DifyClient({ baseUrl: "http://127.0.0.1:1", apiKey: "k" });
    const r = await client.query("x");
    expect(r.degraded).toBe(true);
    expect(r.error).toBe("network-error");
  });

  test("three failures open the circuit and 4th degrades without a hit", async () => {
    mode = "500";
    hits = 0;
    const client = new DifyClient(config, { threshold: 3, resetTimeoutMs: 60_000 });
    for (let i = 0; i < 3; i++) {
      const r = await client.query("x");
      expect(r.degraded).toBe(true);
    }
    expect(client.circuitState).toBe("open");
    const before = hits;
    const r = await client.query("x");
    expect(r.degraded).toBe(true);
    expect(r.error).toBe("circuit-open");
    expect(hits).toBe(before);
  });
});

describe("DifyClient — not configured degrades gracefully", () => {
  test("missing key degrades without throwing", async () => {
    const client = new DifyClient({ baseUrl: "http://x", apiKey: "" });
    const r = await client.query("x");
    expect(r.degraded).toBe(true);
    expect(r.error).toBe("dify-not-configured");
  });
});
