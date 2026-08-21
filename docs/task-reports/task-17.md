# Task 17 Report — Offline Distribution Package and Installers

## Overview

Implementation checkpoint: `3feb08ba0a086af501574ca1b09dc0216add08cc`

Task 17 implements the V1 offline release pipeline for macOS arm64/x64 and Windows x64: cross-compile Codea, acquire the pinned OpenCode v1.18.11 runtime with SHA256 verification, build the self-contained Bun plugin, collect agents/skills/config, generate a strict package manifest, verify package integrity/offline dependency boundaries, produce platform archives, and provide platform installers.

## Delivered

- `packaging/config/release.yaml`
  - Codea 0.1.0
  - OpenCode 1.18.11
  - darwin-arm64 / darwin-x64 / windows-x64
  - SHA256 signing method
- `packaging/scripts/build-runtime.sh`
  - consumes authoritative `runtime/version.json`
  - downloads only the locked platform asset
  - verifies recorded SHA256 before extraction
- `packaging/scripts/build-plugins.sh`
  - `bun test` + `bun run build`
  - self-contained ESM only
  - rejects bare external imports including side-effect imports
- `packaging/scripts/collect-skills.sh`
  - packages Code Reviewer, Unit Test and API Documentation Skills
- `packaging/scripts/build-release.sh`
  - TUI cross-build → runtime → plugin → skills/agents/config → installers → manifest/checksum/offline checks → archive + `.sha256`
- `packaging/scripts/generate-manifest.sh`
  - SHA256 + size for every packaged file
- `packaging/scripts/verify-checksum.sh`
  - manifest schema/safe paths/hash/size/missing/unmanifested-extra fail-closed verification
- `packaging/scripts/verify-offline.sh`
  - rejects plugin package-manager metadata and external package imports
  - checks build-path leakage
  - chains package integrity verification

## Installed runtime wiring remediation

A release-only runtime gap was found while designing the real install smoke: source-tree defaults in `codea` resolve `opencode`, agents, skills and plugin relative to the development tree. A direct installed binary launcher therefore cannot reliably locate package resources under `~/.codea/current` / `%USERPROFILE%\.codea\current`.

This is fixed at the installer/launcher boundary without changing Task 18 or upgrade semantics:

- macOS `~/.codea/bin/codea` is now a launcher script, not a raw binary symlink. It exports:
  - `OPENCODE_BIN=$current/bin/opencode`
  - `CODEA_AGENTS_DIR=$current/agents`
  - `CODEA_SKILLS_DIR=$current/skills`
  - `CODEA_PLUGIN_BUNDLE=$current/plugins/index.js`
- Windows `codea.cmd` binds the same four variables to the `current` Junction before starting `codea.exe`.
- Existing Windows BOM-free `current.txt` + Junction behavior remains unchanged.

`tests/offline/task17_packaging_test.sh` now treats those launcher bindings as required release contracts.

## Fresh release Gate tooling

- `scripts/run-task17-build-gates.sh`
  - requires Go 1.26.5 + Bun
  - builds `darwin-arm64`, `darwin-x64`, `windows-x64`
  - verifies archive and `.sha256`
  - writes `tests/offline/evidence/task17-build-evidence.json`
- `tests/offline/macos_release_smoke.sh`
  - must run on real macOS
  - requires public HTTPS to be blocked
  - installs with package-side `install.sh`
  - verifies `current`, launcher, bundled runtime/plugin/agents/skills
  - starts packaged `opencode serve` and checks `/global/health` reports `1.18.11`
  - launches installed Codea through its real launcher inside a pseudo-terminal and requires it to remain running
  - writes platform evidence JSON
- `tests/offline/windows_release_smoke.ps1`
  - must run on native Windows x64 and explicitly rejects WSL
  - requires public HTTPS to be blocked
  - runs `install.ps1`
  - verifies `current` Junction, launcher and bundled resources
  - starts packaged `opencode.exe serve` and checks health/version
  - launches Codea via `codea.cmd` and requires it to remain running
  - writes `task17-windows-x64-evidence.json`
- `.github/workflows/task16-17-gates.yml`
  - provides an automated runner for three-platform build evidence and Task 16 runtime regression where supported
  - it is not treated as a substitute for the explicitly offline native macOS/Windows install gates

## Verification status

The implementation and Gate definitions are complete enough for the requested re-review, but this report does **not** claim Task 17 A/PASS. The current execution sandbox cannot run native macOS/Windows installers or enforce a genuine no-public-network release environment, and no fresh native evidence has been committed yet.

Required build evidence:

```bash
./scripts/run-task17-build-gates.sh <out> tests/offline/evidence/task17-build-evidence.json
```

Required native macOS evidence (on the matching extracted darwin package, with public network blocked):

```bash
bash tests/offline/macos_release_smoke.sh <extracted-package-dir>
```

Required native Windows x64 evidence (without WSL, with public network blocked):

```powershell
powershell -ExecutionPolicy Bypass -File tests/offline/windows_release_smoke.ps1 -PackageDir <extracted-package-dir>
```

Full regression must also include Task 14/15 agent smoke, Task 16 agent smoke, Go/race/vet/build, plugin tests/build and parity.

Until committed fresh evidence proves the three release builds plus native macOS + Windows x64 install/runtime/offline smoke, Task 17 remains formally `pending` behind Task 16 and is not accepted.

## Scope boundary

Task 17 only covers first-install/offline release packaging. Task 18 transactional upgrade, migration, rollback, crash recovery and version switching remain untouched and pending.
