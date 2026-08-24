# Task 21 Report — Release Parity Certification

## Overview

Task 21 core development is complete. The latest fresh Task 20/21 acceptance gate is green, including real distinct baseline/candidate Runtime parity with **12/12 Required scenarios PASS**.

The remaining work is the final V1 Release Certification closeout. Final certification must aggregate the complete release gate set for one exact source commit and must not claim PASS without real G15 company-intranet mirror evidence.

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

## Remaining final certification

### G15 — company intranet mirrors

G15 must be executed in the approved company intranet environment against the real Maven/npm/PyPI/Go mirrors. GitHub-hosted CI cannot substitute for this evidence.

Until a real G15 PASS artifact exists for the exact source commit being certified:

- automatic preflight may PASS with G15 `deferred`
- final Release Certification must remain blocked
- Task 21 must not be marked fully completed
- Codea V1 must not be declared release-certified

### Final closeout

Once real G15 evidence is available, the final closeout is:

1. assemble G1-G15 plus G2.1/G12.1 for the same source commit;
2. run the strict release certifier;
3. generate the final certification artifact/checklist/report;
4. update `docs/execution-state.yaml`;
5. mark Task 21 and Codea V1 completed only when the strict certification is PASS.

## Current status

**Core development: PASS**  
**Fresh dual Runtime parity: 12/12 PASS**  
**Final Release Certification: BLOCKED ONLY BY REAL G15 COMPANY-INTRANET EVIDENCE**
