# Task 19 Report — Codea Doctor

## Overview

Implementation checkpoint: `b0dd67d4dce8f6af0ee351dbd86ac772eef795b4`

Fresh gate evidence commit: `cc15b5c3d8f2202974eeaef53d5ded750185926c`

Task 19 implements `codea init` and `codea doctor` as the V1 installation/runtime diagnostic path. Doctor is designed to diagnose failures without manufacturing side effects: static checks remain available even when Runtime startup fails, runtime-dependent checks become SKIP/FAIL as appropriate, and public-network probing is not introduced merely for diagnostics.

## Delivered

Core implementation under `tui/internal/doctor/` includes:

- static installation/configuration checks
- Manifest and file SHA256 verification
- configuration schema validation
- permissions validation
- Skill and Agent manifest validation
- Plugin Bundle validation
- OpenCode compatibility/version validation
- Runtime Health check
- enterprise Agent availability check
- model connectivity check
- SSE behavior check
- minimal real inference check
- loopback-only runtime binding check
- Candidate Runtime factory for V2 + C2-temp
- Candidate Doctor adapter used by Task 18 pre/post switch checks

CLI integration provides:

- `codea init` with create-if-absent/idempotent behavior that does not overwrite existing user configuration
- `codea doctor` with `PASS / WARN / FAIL / SKIP` output and non-zero exit status when a FAIL exists
- interrupted-update recovery before normal Doctor execution when a stale transaction marker is present

## Candidate Doctor contract

Task 18 and Task 19 share the same candidate verification path. Candidate Doctor uses the candidate release resources rather than the currently installed runtime:

```text
V2/bin/opencode
+ V2 agents
+ V2 skills
+ V2 plugin
+ C2-temp model/provider configuration
-> isolated Runtime
-> Health / Agent / Model / SSE / Inference
```

Candidate materialization preserves the migrated model/provider configuration instead of overwriting it while registering candidate Agents/Skills/Plugin.

## Fresh acceptance evidence

GitHub Actions workflow:

- Workflow: `Task 18-19 Acceptance Gates`
- Run: `#24`
- Run ID: `32626853054`
- Source commit: `b0dd67d4dce8f6af0ee351dbd86ac772eef795b4`
- Result: **PASS**

Persisted evidence:

- `tests/evidence/task18-19-gate-evidence.json`
- `tests/evidence/candidate-doctor-evidence.json`

Joint Gate confirms:

| Gate | Result |
|---|---|
| Go full regression | PASS |
| Go race | PASS |
| go vet | PASS |
| go build | PASS |
| macOS arm64 cross-build | PASS |
| Windows x64 cross-build | PASS |
| Bun regression/build | PASS |
| Existing Agent Runtime regression | PASS |
| API Documentation Runtime regression | PASS |
| Real Candidate Doctor | PASS |
| Candidate runtime isolation | PASS |
| OpenCode version | PASS — 1.18.11 |

Candidate evidence records:

```text
candidateDoctorPassed = true
openCodeVersion = 1.18.11
phase = pre-switch
runtimeIsolation = true
```

## Final verification remediation

The first Gate run found that the Supervisor Unix implementation was limited to a Darwin-only build tag, so Linux CI could not compile Runtime lifecycle functions. The shared Unix implementation was corrected to non-Windows and the full joint Gate was rerun successfully.

The workflow now also triggers on `supervisor`, `runtime` and `opencode` dependency changes, preventing Doctor/Candidate Runtime regressions from bypassing Task 18/19 acceptance coverage.

## Current status

Task 19 implementation and its fresh joint verification are **PASS**. Formal task status remains `pending` only because Task 18 is the first incomplete task and must receive human acceptance first.

After Task 18 is accepted, Task 19 can be activated and moved to `awaiting_acceptance` using this same fresh checkpoint/evidence provided no Task 19-relevant code changes occur in between.

Task 20 remains pending and has not been started.
