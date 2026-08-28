# Task 27 Report — Windows Trust Chain & Installed Runtime Reliability

Task 27 production implementation, automated verification, and human acceptance are complete.

Production implementation checkpoint: `0390d4697cfee3832d0fa3703bbb35a85f4df2f3`

## Scope delivered

Task 27 closes the Windows trust/reliability gaps without changing the Codea Runtime abstraction:

- Windows Runtime startup keeps the bounded `ERROR_ACCESS_DENIED` retry and minimal process-rights Job Object model;
- formal Windows package installation verifies manifest SHA256/size before install and removes Mark-of-the-Web only from the verified installed copy;
- a native Windows installed-package lifecycle Gate exercises fresh install, MOTW install, `CODEA_HOME` containing spaces, forward upgrade, rollback, and reselect upgrade using the real packaged `codea.exe` / bundled OpenCode `v1.18.11`;
- Windows release signing remains fail-closed and uses real `signtool.exe` SHA256 Authenticode signing plus `Get-AuthenticodeSignature` verification;
- CI mechanical proof uses an ephemeral two-level trust chain (`Root CA -> code-signing leaf`) generated on Ubuntu, then validates the signed executable as `Valid` on a Windows hosted runner;
- the same temporary signing identity drives a full Stable Finalized Release E2E through `finalize-release.ps1 -Channel stable`;
- deterministic Windows security evidence records final release SHA256, Codea/OpenCode identity and the finalized signer status/subject/thumbprint;
- Task 26 native Windows regression is rerun after all Task 27 lifecycle/signing/finalization checks.

## TDD / blocker remediation evidence

During fresh Windows execution, the Gate exposed and corrected concrete test/release blockers rather than weakening verification:

1. Authenticode proof identity was changed from self-signed peer trust to a real temporary `Root CA -> code-signing leaf` chain so Windows `WinVerifyTrust` / `Get-AuthenticodeSignature` returns `Valid`.
2. `verify-signature.ps1` PowerShell interpolation was made parse-safe (`${resolved}:`).
3. lifecycle diagnostics were made parse-safe (`${Scenario}:`).
4. lifecycle helper parameters no longer shadow PowerShell's read-only `$HOME`; they use `$CodeaHome`.
5. installed shim health checks invoke `codea doctor` through the installed `bin` directory on `PATH`, matching real user usage and validating `CODEA_HOME` paths containing spaces.
6. Human review found that the prior Task 27 Gate signed only an extracted `codea.exe`, not the final release archive. RED run `33178564371` failed the new Authenticode contract until a real Stable Finalized Release E2E was wired.

No Authenticode validity requirement, Runtime health assertion, process cleanup requirement, lifecycle scenario, MOTW self-heal, Installer Unblock, or bounded access-denied retry was relaxed.

## Stable Finalized Release E2E

Fresh implementation head `0390d4697cfee3832d0fa3703bbb35a85f4df2f3`, Task 27 Gate run `33178843283`: **PASS**.

The Windows Gate proves the complete chain:

```text
formal unsigned Windows ZIP
→ stable without credentials FAIL-CLOSED
→ ephemeral Root CA / code-signing leaf PFX
→ finalize-release.ps1 -Channel stable
→ SignTool signs final codea.exe
→ Authenticode verification = Valid
→ manifest.json rebinds bin/codea.exe SHA256 + size
→ final ZIP rebuilt
→ final ZIP .sha256 regenerated and verified
→ security evidence records Valid signer subject/thumbprint
→ finalized signed ZIP install.ps1
→ codea doctor
→ OpenCode 1.18.11 Runtime Health PASS
```

The negative stable test confirms missing signing credentials fail before finalization: no evidence file is generated, no finalized SHA256 sidecar is generated, and the unsigned archive hash is unchanged.

The positive E2E validates on the finalized ZIP:

- extracted `bin/codea.exe` has `Get-AuthenticodeSignature = Valid`;
- signer thumbprint equals the temporary code-signing leaf;
- `manifest.json` `bin/codea.exe` SHA256 and size equal the final signed executable bytes;
- `<release>.zip.sha256` equals the actual finalized ZIP SHA256;
- security evidence has `signatureStatus = Valid`, non-empty `signerSubject`, non-empty matching `signerThumbprint`, and the final ZIP `releaseSha256`;
- the finalized signed ZIP passes the complete fresh/MOTW/spaces/upgrade/rollback installed-package lifecycle and real OpenCode `v1.18.11` health checks.

## Fresh exact-head automated evidence

Implementation run `33178843283`: **SUCCESS**

- Task 27 trust contract: PASS
- Task 27 Authenticode contract: PASS
- Task 27 security evidence contract: PASS
- Formal Windows x64 release package build: PASS
- Ephemeral Root/leaf Authenticode identity generation: PASS
- Full native Windows Go regression: PASS
- bounded access-denied + process-tree regression: PASS
- standalone real SignTool + Authenticode proof: PASS / Valid
- Stable without signing credentials fail-closed: PASS
- **Stable Finalized Release E2E: PASS**
- finalized 360 / EDR evidence: PASS / Valid signer
- finalized signed package lifecycle (`fresh / MOTW / spaces / upgrade / rollback`): PASS
- final Task 26 native Windows regression: PASS

Administrative closeout head `246837585f7d68ecbc867e455f0eceb4a70a1a3a`, Task 27 Gate run `33179384852`: **SUCCESS**.

## Human acceptance and integration

- Accepted: **YES — 2026-08-28**
- Automated verification: **PASS**
- Task Gate: **PASS**
- Implementation checkpoint: `0390d4697cfee3832d0fa3703bbb35a85f4df2f3`
- Fresh implementation Gate: `33178843283` — **PASS**
- Fresh administrative closeout Gate: `33179384852` — **PASS**
- Integrated to `main` through PR #1
- Merge commit: `09404791f221d803c278210a57a99cf471aea14d`

Current status: **COMPLETED**
