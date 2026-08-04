# Task 1 Acceptance Blockers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the three Task 1 acceptance blockers with executable, independently reproducible evidence.

**Architecture:** Keep the macOS-only S1 runner responsible for network isolation, but move evidence evaluation into a deterministic validator that can be regression-tested with fixtures. Add one reusable Skill-isolation runner that creates its own fixture tree and records raw `/skill` responses for S5/S6. Lock the exact official v1.18.11 CLI release assets and SHA-256 digests for every V1 platform in `runtime/version.json`.

**Tech Stack:** Bash, Python 3 standard library, OpenCode v1.18.11 HTTP API, GitHub Release asset digests, existing execution-state validator.

## Global Constraints

- Continue Task 1 only; do not start Task 2.
- Preserve the existing S1 raw evidence and add regression coverage before changing validation behavior.
- S5/S6 evidence must be reproducible from committed scripts and fixtures and include unmodified raw `/skill` JSON.
- Release checksums must name exact assets and contain no placeholders.
- Use only fast-forward pushes to `develop`.

---

### Task 1: Make S1 failures return non-zero

**Files:**
- Create: `scripts/check-s1-offline-evidence.sh`
- Modify: `docs/spike-artifacts/s1-network-test.sh`
- Test: `tests/phase0/check_s1_offline_evidence_test.sh`

**Interfaces:**
- Consumes: an S1 result directory containing `health.json`, `opencode-internal.log`, and `traffic-*.pcap`.
- Produces: exit 0 only when health is true, no forbidden host or ERROR appears, and no DNS/HTTP/HTTPS traffic is detected; otherwise exit 1.

- [x] Write fixture-driven tests for forbidden host, ERROR, abnormal traffic, missing evidence, and clean evidence.
- [x] Run the test and verify it fails because the validator does not exist.
- [x] Implement the minimum validator and invoke it from the S1 runner.
- [x] Run the test and shell syntax checks until green.

### Task 2: Make S5/S6 independently reproducible

**Files:**
- Create: `scripts/run-skill-isolation-spikes.sh`
- Create: `tests/fixtures/skill-isolation/**/SKILL.md`
- Create: `tests/phase0/run_skill_isolation_spikes_test.sh`
- Create: `docs/spike-artifacts/s5-s6-20260803-rerun/*.json`
- Modify: `docs/spike-report.md`
- Modify: `docs/task-reports/task-01.md`

**Interfaces:**
- Consumes: `OPENCODE_BIN` and an output directory.
- Produces: health responses, complete raw `/skill` responses, derived name lists, Runtime logs, copied fixture manifest, and a non-zero exit when any profile differs from its exact expected set.

- [x] Write a fake-Runtime test that exercises the runner and fails while it is absent.
- [x] Implement fixture creation, profile startup, raw response capture, exact-set assertions, and cleanup.
- [x] Run the test, then run the same script against the real locked Runtime.
- [x] Verify every raw response is valid JSON and mechanically derives the committed name list.

### Task 3: Lock all V1 release asset hashes

**Files:**
- Modify: `runtime/version.json`
- Modify: `docs/spike-artifacts/s1-release.json`
- Create: `tests/phase0/version_lock_test.sh`
- Modify: `docs/codea-v1-handoff.md`
- Modify: `docs/spike-report.md`
- Modify: `docs/task-reports/task-01.md`

**Interfaces:**
- Consumes: official GitHub v1.18.11 Release metadata and downloaded CLI assets.
- Produces: exact asset name, size, SHA-256, and source URL for Linux x64, macOS arm64/x64, and Windows x64.

- [x] Write a regression test that rejects placeholders, missing assets, malformed digests, and duplicate platform assets.
- [x] Run it and verify the current `TBD-by-spike` values fail.
- [x] Download the three missing official assets to `/tmp`, calculate SHA-256, and compare against official Release digests.
- [x] Update version/evidence/report files and run the regression test until green.

### Task 4: Re-evaluate Task 1 gate

**Files:**
- Modify: `docs/execution-state.yaml`
- Modify: `docs/codea-v1-handoff.md`
- Modify: `docs/task-reports/task-01.md`

**Interfaces:**
- Consumes: all Task 1 tests and evidence.
- Produces: a truthful `awaiting_acceptance` state only if every fresh verification exits 0.

- [x] Run Go 1.26.5 tests/build, all Phase 0 shell tests, real Phase 0 gate, JSON checks, shell syntax, state validation, credential scan, and `git diff --check`.
- [ ] Commit the evidence checkpoint, update checkpoint references to that real commit, and commit the state/report closure.
- [ ] Fetch and fast-forward push `develop`; verify remote SHA equals local SHA.
