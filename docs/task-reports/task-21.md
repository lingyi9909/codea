# Task 21 Report — Release Parity Certification

## Overview

Task 21 core development and final closeout automation are complete. The latest fresh Task 20/21 acceptance gate before the closeout-only changes is green, including real distinct baseline/candidate Runtime parity with **12/12 Required scenarios PASS**.

The only external evidence still missing is G15 from the approved company intranet. Final certification must aggregate the complete release gate set for one exact source commit and must not claim PASS without real G15 company-intranet mirror evidence.

## Completed core work

- Release gate evidence format and merge tooling
- Native macOS/Windows offline gate derivation for G1/G2/G2.1
- Core release gates G3-G14
- Real locked OpenCode v1.18.11 runtime smoke coverage
- Real dual Runtime release parity G12.1
- Distinct vanilla baseline vs Codea candidate execution
- 12/12 Required task-effect parity PASS
- Strict release-certifier path and GitHub release-certification workflow
- Automatic GitHub preflight that keeps G15 deferred because GitHub cannot access approved company intranet mirrors
- Windows G15 intranet mirror runner for real-environment evidence collection

## Final closeout automation

The final closeout path is now machine-driven instead of manually assembled:

1. `scripts/merge-release-gates.py` requires the exact 17-gate set: G1-G15 plus G2.1/G12.1, all bound to one source commit.
2. `tui/cmd/release-certifier` performs the strict final certification and refuses a deferred/failed gate.
3. `scripts/generate-release-closeout.py` accepts only a passed strict certification artifact with **17/17 gate PASS** and **12/12 Runtime parity PASS**.
4. Only after those checks does it generate:
   - `release-certification-checklist.md`
   - `release-certification-report.md`
5. `.github/workflows/release-certification.yml` runs the generator after the strict certifier and uploads the checklist/report together with the machine evidence.
6. Regression coverage explicitly rejects G15 `deferred` and verifies no final certification documents are written in that case.

Therefore the repository-side Release Certification closeout implementation is complete. The final workflow execution remains intentionally blocked until real G15 evidence exists.

## Remaining external certification evidence

### G15 — company intranet mirrors

G15 must be executed in the approved company intranet environment against the real Maven/npm/PyPI/Go mirrors. GitHub-hosted CI cannot substitute for this evidence.

Until a real G15 PASS artifact exists for the exact source commit being certified:

- automatic preflight may PASS with G15 `deferred`
- final Release Certification must remain blocked
- Task 21 must not be marked fully completed
- Codea V1 must not be declared release-certified
- the final closeout generator must not emit a RELEASE CERTIFIED checklist/report

### Final execution after G15

Once real G15 evidence is available for the final source commit:

1. run `scripts/run-g15-intranet-gate.ps1` inside the approved company intranet;
2. base64-encode the produced G15 gate artifact;
3. dispatch `V1 Release Certification` with that exact `source_commit` and `g15_gate_b64`;
4. require strict certifier PASS;
5. obtain the uploaded machine evidence plus final checklist/report;
6. update `docs/execution-state.yaml` and mark Task 21 / Codea V1 completed only after the strict final artifact is reviewed.

## Current status

**Core development: PASS**  
**Repository-side final closeout automation: COMPLETE**  
**Fresh dual Runtime parity baseline: 12/12 PASS**  
**Final Release Certification: BLOCKED ONLY BY REAL G15 COMPANY-INTRANET EVIDENCE**
