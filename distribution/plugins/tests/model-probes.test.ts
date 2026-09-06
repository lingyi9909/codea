import { describe, expect, test } from "bun:test";
import { ProbeAttemptState } from "../src/tools/probe-state";
import { createProbeToolCallTool } from "../src/tools/probe-tool-call";
import { createProbeStructuredTool } from "../src/tools/probe-structured";
import { createProbePatchTool } from "../src/tools/probe-patch";
import { createProbePlanTool } from "../src/tools/probe-plan";

function ctx(sessionId = "session-a"): any {
  return {
    sessionId,
    rootTurnId: "",
    agent: "model-evaluator",
    projectRoot: "/project",
    audit: {},
    guard: { after: () => {} },
  };
}

const exactStructured = {
  items: [
    { id: "A", enabled: true },
    { id: "B", enabled: false },
  ],
  summary: { count: 2 },
};

const exactPatch = {
  original: "alpha\nbeta\n",
  instruction: "replace beta with gamma",
  result: "alpha\ngamma\n",
};

const exactPlan = {
  steps: [
    { id: "1", action: "inspect" },
    { id: "2", action: "edit" },
    { id: "3", action: "verify" },
  ],
};

describe("model qualification probes", () => {
  test("accept exact public contracts and emit bounded safe evidence", async () => {
    const state = new ProbeAttemptState();
    const cases = [
      [createProbeToolCallTool(state), { token: "CODEA-17", value: 42 }, "tool_call"],
      [createProbeStructuredTool(state), exactStructured, "structured"],
      [createProbePatchTool(state), exactPatch, "patch"],
      [createProbePlanTool(state), exactPlan, "planning"],
    ] as const;

    for (const [tool, input, probe] of cases) {
      const result = await tool.execute(input, ctx(`session-${probe}`));
      expect(result.ok).toBe(true);
      if (!result.ok) throw new Error(result.error.message);
      expect(result.data).toEqual({
        acknowledgement: "PASS",
        codeaProbe: probe,
        codeaProbeResult: "pass",
        codeaProbeAttempt: "1",
      });
    }
  });

  test("reject missing extra and wrong values", async () => {
    const factories = [
      () => createProbeToolCallTool(new ProbeAttemptState()),
      () => createProbeStructuredTool(new ProbeAttemptState()),
      () => createProbePatchTool(new ProbeAttemptState()),
      () => createProbePlanTool(new ProbeAttemptState()),
    ];
    const invalid = [
      [{ token: "CODEA-17" }, { token: "CODEA-17", value: 42, extra: true }, { token: "wrong", value: 42 }],
      [{ items: exactStructured.items }, { ...exactStructured, extra: true }, { ...exactStructured, summary: { count: 3 } }],
      [{ original: exactPatch.original }, { ...exactPatch, extra: true }, { ...exactPatch, result: "alpha\nbeta\n" }],
      [{ steps: exactPlan.steps.slice(0, 2) }, { ...exactPlan, extra: true }, { steps: [...exactPlan.steps].reverse() }],
    ];

    for (let i = 0; i < factories.length; i++) {
      for (const input of invalid[i]) {
        const result = await factories[i]().execute(input, ctx(`invalid-${i}`));
        expect(result.ok).toBe(false);
      }
    }
  });

  test("limits each probe to two attempts per runtime session", async () => {
    const state = new ProbeAttemptState();
    const tool = createProbeToolCallTool(state);
    const input = { token: "CODEA-17", value: 42 };

    const first = await tool.execute(input, ctx("bounded"));
    const second = await tool.execute(input, ctx("bounded"));
    const third = await tool.execute(input, ctx("bounded"));
    expect(first.ok && first.data.codeaProbeAttempt).toBe("1");
    expect(second.ok && second.data.codeaProbeAttempt).toBe("2");
    expect(third.ok).toBe(false);
    if (third.ok) throw new Error("third probe unexpectedly passed");
    expect(third.error.message).toContain("PROBE_ATTEMPT_LIMIT");

    const otherSession = await tool.execute(input, ctx("other-session"));
    expect(otherSession.ok && otherSession.data.codeaProbeAttempt).toBe("1");
  });
});
