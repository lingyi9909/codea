# Codea V1 Release Certification Checklist

## Certification baseline

This checklist is the human-review companion for Task 21. The machine-verifiable final artifacts are produced by `.github/workflows/release-certification.yml` and must all bind to one exact `sourceCommit`.

**Current release state:** NOT YET RELEASE CERTIFIED.

Task 21 core development and the latest fresh automated gate are green, including real distinct baseline/candidate Runtime parity with **12/12 Required scenarios PASS**. Final certification remains blocked until G15 is executed against the approved company-intranet Maven/npm/PyPI/Go mirrors and produces real PASS evidence for the exact source commit being certified.

## Required release gates

| Gate | Current state | Final requirement |
| --- | --- | --- |
| G1 | automated evidence available | PASS for final `sourceCommit` |
| G2 | automated evidence available | PASS for final `sourceCommit` |
| G2.1 | automated evidence available | PASS for final `sourceCommit` |
| G3 | automated evidence available | PASS for final `sourceCommit` |
| G4 | automated evidence available | PASS for final `sourceCommit` |
| G5 | automated evidence available | PASS for final `sourceCommit` |
| G6 | automated evidence available | PASS for final `sourceCommit` |
| G7 | automated evidence available | PASS for final `sourceCommit` |
| G8 | automated evidence available | PASS for final `sourceCommit` |
| G9 | automated evidence available | PASS for final `sourceCommit` |
| G10 | automated evidence available | PASS for final `sourceCommit` |
| G11 | automated evidence available | PASS for final `sourceCommit` |
| G12 | automated evidence available | PASS for final `sourceCommit` |
| G12.1 | 12/12 Required Runtime parity PASS in latest fresh gate | PASS for final `sourceCommit` |
| G13 | automated evidence available | PASS for final `sourceCommit` |
| G14 | automated evidence available | PASS for final `sourceCommit` |
| G15 | **BLOCKED — company intranet required** | real approved-company-intranet mirror PASS evidence for final `sourceCommit` |

## G15 evidence requirements

G15 is not allowed to be satisfied by GitHub-hosted CI, mocked endpoints, public mirrors, copied/stale artifacts, or manually edited JSON.

The evidence must be produced in the approved company intranet by the repository G15 runner and must verify the real configured mirrors for:

- Maven
- npm
- PyPI
- Go modules

The resulting G15 gate artifact must have `status: pass`, non-empty evidence, and a `sourceCommit` exactly equal to the commit used by every other final release gate.

## Final certification procedure

- [x] Task 21 core release-certification tooling implemented.
- [x] Latest fresh Task 20/21 acceptance gate is green.
- [x] Real distinct baseline/candidate Runtime parity is 12/12 Required scenarios PASS.
- [x] Automatic GitHub preflight keeps G15 deferred instead of fabricating intranet evidence.
- [x] Strict final certifier rejects non-PASS release gates.
- [x] Final checklist/report generator rejects deferred or failed G15.
- [ ] Select and freeze the exact final `sourceCommit` to certify.
- [ ] Run G15 inside the approved company intranet for that exact `sourceCommit`.
- [ ] Collect real G15 PASS artifact from `run-g15-intranet-gate.ps1`.
- [ ] Run `V1 Release Certification` by `workflow_dispatch` with the exact `source_commit` and the real G15 artifact.
- [ ] Confirm merged `release-gates.json` contains exactly G1–G15 plus G2.1/G12.1, all PASS, all bound to the same `sourceCommit`.
- [ ] Confirm `release-certification.json` has `passed: true`.
- [ ] Confirm final generated `release-certification-checklist.md` and `release-certification-report.md` identify the same `sourceCommit` and 12/12 Required Runtime parity PASS.
- [ ] Update `docs/execution-state.yaml` only after the strict final certification artifact is PASS and reviewed.
- [ ] Mark Task 21 completed and Codea V1 release-certified only after all items above are complete.

## Fail-closed rules

Any of the following keeps the release blocked:

- G15 missing, deferred, failed, stale, or from a non-approved environment;
- any gate missing or not PASS;
- mixed `sourceCommit` values across gate artifacts;
- Runtime Required parity below 12/12;
- strict certifier failure;
- final checklist/report not generated from the strict machine certification artifact.

Until every final item is satisfied, the only valid project status is: **Task 21 in progress / final Release Certification blocked on remaining evidence**.
