---
name: debug
description: Evidence-driven enterprise debugging with controlled fixes and fresh verification.
---

# Debug

Use this skill for application failures, test failures, build failures, regressions, and unexpected runtime behavior.

## Required loop

1. **Collect evidence** from the reported failure, logs, stack traces, tests, code, and workspace state. Separate facts from assumptions.
2. **Reproduce** the problem with the smallest relevant check when feasible. If it cannot be reproduced, record why and what evidence is still available.
3. **Root cause** the failure by tracing backward from the symptom to the earliest supported cause. Prefer fixing the cause over masking the symptom.
4. **Controlled fix** with the smallest scoped change. Mutating files or executing commands remains subject to Codea approval, project path policy, command policy, DLP, audit, and offline constraints.
5. **Fresh verification** after the change: rerun the focused failing check, then the relevant regression checks.
6. Summarize evidence, root cause, files changed, verification performed, and remaining risks.

Do not claim success merely because code was edited. A fix is only complete when fresh verification supports it. If approval or security policy blocks a required action, report the block and preserve the controls.
