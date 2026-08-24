# Task 21 Report — Release Parity Certification

## Overview

Task 21 implementation is complete and has no code blocker. The final missing process condition was a fresh Task 20/21 acceptance Gate on the latest exact-release-parity checkpoint.

That condition is now satisfied:

- Workflow: `Task 20-21 Acceptance Gates`
- Run: `#77`
- Run ID: `32719737844`
- Source commit: `62c0c7038a6324c115a1be764379e83b21898090`
- Result: **PASS**
- Real distinct baseline/candidate Runtime parity: **12/12 Required scenarios PASS**

G15 remains `deferred` by explicit acceptance decision. It does **not** block Task 21 acceptance.

## Completed core work

- Release gate evidence format and merge tooling
- Native macOS/Windows offline gate derivation for G1/G2/G2.1
- Core release gates G3-G14
- Real locked OpenCode v1.18.11 runtime smoke coverage
- Real dual Runtime release parity G12.1
- Distinct vanilla baseline vs Codea candidate execution
- 12/12 Required task-effect parity PASS
- Strict release-certifier path and GitHub release-certification workflow
- Exact 12/12 parity requirement enforced by the release certifier
- Automatic GitHub preflight that keeps G15 deferred because GitHub cannot access approved company intranet mirrors
- Windows G15 intranet mirror runner for future real-environment evidence collection
- Machine-driven final closeout generator for certification checklist/report when a strict release certification is eventually executed

## Fresh Gate verification — run #77

The fresh Gate executed against exact source commit `62c0c7038a6324c115a1be764379e83b21898090` and completed successfully.

Verified steps include:

- Bun plugin regression/build — PASS
- Release evidence tool regression — PASS
- Go full regression — PASS
- Locked OpenCode v1.18.11 download/checksum — PASS
- Real distinct baseline/candidate parity — PASS
- Runtime parity result — **12/12**
- Parity evidence artifact upload — PASS

Artifact:

`task20-21-release-parity-62c0c7038a6324c115a1be764379e83b21898090`

## G15 acceptance boundary

G15 remains deferred because it requires the approved company intranet and real Maven/npm/PyPI/Go mirrors.

The formal acceptance decision for this round is:

- G15 stays `deferred`;
- G15 is not substituted with GitHub-hosted evidence;
- G15 does **not** block Task 21 acceptance;
- future strict V1 Release Certification may still consume real G15 evidence when that separate release-certification activity is performed.

This separates **Task 21 acceptance** from the optional later completion of the real-intranet G15 release evidence.

## Final acceptance

Formal human acceptance recorded on 2026-08-24:

- **Implementation: A — PASS, no code blocker**
- **Process acceptance: the only B-condition was fresh Gate on `62c0...`; run #77 is PASS, so the condition is satisfied**
- **G15: deferred and explicitly non-blocking for Task 21 acceptance**
- **Task 21: final acceptance PASS**

## Current status

**Task 21: COMPLETED / ACCEPTED**  
**Fresh Gate #77: PASS**  
**Real dual Runtime parity: 12/12 PASS**  
**G15: DEFERRED — NON-BLOCKING FOR TASK 21**
