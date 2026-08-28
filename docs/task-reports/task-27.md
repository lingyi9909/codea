# Task 27 Report — Windows Trust Chain & Installed Runtime Reliability

Task 27 production implementation and automated verification are complete. Human acceptance is still pending.

Production implementation checkpoint: `29b6196c0e949548539c64ea61a022cf5e210dc9`

## Scope delivered

Task 27 closes the Windows trust/reliability gaps without changing the Codea Runtime abstraction:

- Windows Runtime startup keeps the bounded `ERROR_ACCESS_DENIED` retry and minimal process-rights Job Object model;
- formal Windows package installation verifies manifest SHA256/size before install and removes Mark-of-the-Web only from the verified installed copy;
- a native Windows installed-package lifecycle Gate exercises fresh install, MOTW install, `CODEA_HOME` containing spaces, forward upgrade, rollback, and reselect upgrade using the real packaged `codea.exe` / bundled OpenCode `v1.18.11`;
- Windows release signing remains fail-closed and uses real `signtool.exe` SHA256 Authenticode signing plus `Get-AuthenticodeSignature` verification;
- CI mechanical proof uses an ephemeral two-level trust chain (`Root CA -> code-signing leaf`) generated on Ubuntu, then validates the signed executable as `Valid` on a Windows hosted runner;
- deterministic Windows security evidence records release SHA256, Codea version, locked OpenCode version/checksum, and signature metadata;
- Task 26 native Windows regression is rerun after all Task 27 lifecycle/signing checks.

## TDD / blocker remediation evidence

During fresh Windows execution, the Gate exposed and corrected concrete test/release blockers rather than weakening verification:

1. Authenticode proof identity was changed from self-signed peer trust to a real temporary `Root CA -> code-signing leaf` chain so Windows `WinVerifyTrust` / `Get-AuthenticodeSignature` returns `Valid`.
2. `verify-signature.ps1` PowerShell interpolation was made parse-safe (`${resolved}:`).
3. lifecycle diagnostics were made parse-safe (`${Scenario}:`).
4. lifecycle helper parameters no longer shadow PowerShell's read-only `$HOME`; they use `$CodeaHome`.
5. installed shim health checks now invoke `codea doctor` through the installed `bin` directory on `PATH`, matching real user usage and correctly validating `CODEA_HOME` paths containing spaces.

No Authenticode validity requirement, Runtime health assertion, process cleanup requirement, or lifecycle scenario was relaxed.

## Fresh exact-head automated evidence

Implementation head `29b6196c0e949548539c64ea61a022cf5e210dc9`:

- `Task 27 Windows Trust Chain Gates` run `33154929611`: **PASS**
- Contract job: **PASS**
  - Task 27 trust contract: PASS
  - Task 27 Authenticode contract: PASS
  - Task 27 security evidence contract: PASS
- Formal Windows x64 release package build: **PASS**
- Ephemeral Root/leaf Authenticode identity generation on Ubuntu: **PASS**
- Full native Windows Go regression: **PASS**
- bounded access-denied + process-tree regression: **PASS**
- prepare ephemeral Authenticode proof material: **PASS**
- real SignTool signing: **PASS**
- `Get-AuthenticodeSignature` verification: **PASS / Valid**
- deterministic 360 / EDR evidence generation: **PASS**
- installed-package real lifecycle (`fresh install / MOTW / spaces / upgrade / rollback`): **PASS**
- final Task 26 native Windows regression: **PASS**

The lifecycle Gate emits `TASK27_WINDOWS_INSTALLED_PACKAGE_REAL_LIFECYCLE PASS` only after all scenarios complete successfully.

## Human acceptance

- Accepted: **NO — pending user review**
- Automated verification: **PASS**
- Task Gate: **PASS**
- Implementation checkpoint: `29b6196c0e949548539c64ea61a022cf5e210dc9`
- Fresh implementation Gate: `33154929611` — **PASS**

Current status: **AWAITING ACCEPTANCE**
