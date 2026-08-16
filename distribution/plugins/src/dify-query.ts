// Dify query with circuit breaker + graceful degradation.
//
// Consecutive failure threshold -> circuit open -> immediate degradation ->
// timeout -> half-open probe -> success closes / failure re-opens.
//
// Dify is an intranet service: no public-network fallback, unavailability must
// never crash the Codea runtime, prompts are never logged, and the API key is
// supplied only via environment variables (never a config file).

export type CircuitState = "closed" | "open" | "half-open";

export interface CircuitBreakerOptions {
  threshold: number;
  resetTimeoutMs: number;
  now?: () => number;
}

export class CircuitOpenError extends Error {
  constructor() {
    super("circuit open");
    this.name = "CircuitOpenError";
  }
}

export class DifyHttpError extends Error {
  constructor(readonly status: number) {
    super(`dify http ${status}`);
    this.name = "DifyHttpError";
  }
}

export class DifyInvalidResponseError extends Error {
  constructor() {
    super("dify invalid response");
    this.name = "DifyInvalidResponseError";
  }
}

export class CircuitBreaker {
  private state: CircuitState = "closed";
  private consecutiveFailures = 0;
  private openedAt = 0;
  private halfOpenProbeInFlight = false;
  private readonly opts: CircuitBreakerOptions;

  constructor(opts: Partial<CircuitBreakerOptions> = {}) {
    this.opts = {
      threshold: opts.threshold ?? 3,
      resetTimeoutMs: opts.resetTimeoutMs ?? 60_000,
      now: opts.now ?? (() => Date.now()),
    };
  }

  get currentState(): CircuitState {
    this.tryTransitionToHalfOpen();
    return this.state;
  }

  private get now(): number {
    return this.opts.now!();
  }

  private tryTransitionToHalfOpen(): void {
    if (this.state === "open" && this.now - this.openedAt >= this.opts.resetTimeoutMs) {
      this.state = "half-open";
      this.halfOpenProbeInFlight = false;
    }
  }

  async execute<T>(fn: () => Promise<T>): Promise<T> {
    this.tryTransitionToHalfOpen();
    if (this.state === "open") {
      throw new CircuitOpenError();
    }
    if (this.state === "half-open" && this.halfOpenProbeInFlight) {
      throw new CircuitOpenError();
    }
    if (this.state === "half-open") {
      this.halfOpenProbeInFlight = true;
    }
    try {
      const value = await fn();
      this.onSuccess();
      return value;
    } catch (err) {
      this.onFailure();
      throw err;
    }
  }

  private onSuccess(): void {
    this.state = "closed";
    this.consecutiveFailures = 0;
    this.halfOpenProbeInFlight = false;
  }

  private onFailure(): void {
    this.consecutiveFailures += 1;
    this.halfOpenProbeInFlight = false;
    if (this.state === "half-open" || this.consecutiveFailures >= this.opts.threshold) {
      this.state = "open";
      this.openedAt = this.now;
    }
  }
}

export interface DifyConfig {
  baseUrl: string;
  apiKey: string;
  timeoutMs?: number;
}

export interface DifyQueryResult {
  degraded: boolean;
  answer?: string;
  error?: string;
}

export class DifyClient {
  private readonly breaker: CircuitBreaker;
  private readonly config: DifyConfig;

  constructor(config: DifyConfig, breakerOpts?: Partial<CircuitBreakerOptions>) {
    this.config = config;
    this.breaker = new CircuitBreaker(breakerOpts);
  }

  get circuitState(): CircuitState {
    return this.breaker.currentState;
  }

  async query(question: string): Promise<DifyQueryResult> {
    if (!this.config.baseUrl || !this.config.apiKey) {
      return { degraded: true, error: "dify-not-configured" };
    }
    try {
      const answer = await this.breaker.execute(() => this.fetchQuery(question));
      return { degraded: false, answer };
    } catch (err) {
      if (err instanceof CircuitOpenError) {
        return { degraded: true, error: "circuit-open" };
      }
      return { degraded: true, error: classifyDifyError(err) };
    }
  }

  private async fetchQuery(question: string): Promise<string> {
    const timeoutMs = this.config.timeoutMs ?? 10_000;
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), timeoutMs);
    try {
      const res = await fetch(`${this.config.baseUrl}/chat-messages`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${this.config.apiKey}`,
        },
        body: JSON.stringify({
          inputs: {},
          query: question,
          response_mode: "blocking",
          user: "codea",
        }),
        signal: controller.signal,
      });
      if (!res.ok) {
        throw new DifyHttpError(res.status);
      }
      let data: unknown;
      try {
        data = await res.json();
      } catch {
        throw new DifyInvalidResponseError();
      }
      const answer = (data as { answer?: unknown })?.answer;
      if (typeof answer !== "string") {
        throw new DifyInvalidResponseError();
      }
      return answer;
    } finally {
      clearTimeout(timer);
    }
  }
}

export function classifyDifyError(err: unknown): string {
  const name = (err as Error)?.name ?? "";
  if (name === "AbortError" || name === "TimeoutError") return "timeout";
  if (err instanceof DifyHttpError) {
    return err.status >= 500 ? "http-5xx" : "http-4xx";
  }
  if (err instanceof DifyInvalidResponseError) return "invalid-response";
  return "network-error";
}

export function difyConfigFromEnv(env: Record<string, string | undefined> = process.env): DifyConfig | null {
  const baseUrl = env["DIFY_BASE_URL"];
  const apiKey = env["DIFY_API_KEY"];
  if (!baseUrl || !apiKey) return null;
  return { baseUrl, apiKey };
}
