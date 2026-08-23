# Task 17 Report — Offline Distribution Package and Installers

## Overview

Implementation checkpoint: `3feb08ba0a086af501574ca1b09dc0216add08cc`

Task 17 implements the V1 offline release pipeline for macOS arm64/x64 and Windows x64: cross-compile Codea, acquire the pinned OpenCode v1.18.11 runtime with SHA256 verification, build the self-contained Bun plugin, collect agents/skills/config, generate a strict package manifest, verify package integrity/offline dependency boundaries, produce platform archives, and provide platform installers.

## Delivered

- `packaging/config/release.yaml`
- `packaging/scripts/build-runtime.sh`
- `packaging/scripts/build-plugins.sh`
- `packaging/scripts/collect-skills.sh`
- `packaging/scripts/build-release.sh`
- `packaging/scripts/generate-manifest.sh`
- `packaging/scripts/verify-checksum.sh`
- `packaging/scripts/verify-offline.sh`
- `packaging/platform/macos/install.sh`
- `packaging/platform/windows/install.ps1`
- `scripts/run-task17-build-gates.sh`
- `tests/offline/macos_release_smoke.sh`
- `tests/offline/windows_release_smoke.ps1`

## Installed runtime wiring remediation

Installed Codea must not depend on source-tree relative paths. The macOS and Windows launchers explicitly bind the installed `current` version through:

```text
OPENCODE_BIN
CODEA_AGENTS_DIR
CODEA_SKILLS_DIR
CODEA_PLUGIN_BUNDLE
```

macOS resolves these under `~/.codea/current`; Windows resolves them under the `current` Junction. This keeps the installed binary, bundled OpenCode runtime, agents, skills and plugin on the same installed version.

## Task 17 acceptance evidence

Task 17 owns the release-specific gates; they do not block Task 16.

### Gate status

| Gate | Status |
|------|--------|
| task17PackagingContract | pass |
| task17ThreePlatformBuildEvidence | pass |
| task17MacOSNativeOfflineEvidence | deferred |
| task17WindowsNativeOfflineEvidence | deferred |

### Required before Task 17 final acceptance

- full regression relevant to the release contents
- three-platform release build:
  - `darwin-arm64`
  - `darwin-x64`
  - `windows-x64`
- archive/checksum/integrity/offline static verification

Build evidence command:

```bash
./scripts/run-task17-build-gates.sh <out> tests/offline/evidence/task17-build-evidence.json
```

### Deferred native environment evidence

The following are explicitly **deferred**, not PASS:

```text
task17MacOSNativeOfflineEvidence: deferred
task17WindowsNativeOfflineEvidence: deferred
```

They must only be marked PASS after actual native execution:

macOS:

```bash
bash tests/offline/macos_release_smoke.sh <extracted-package-dir>
```

Windows x64, no WSL:

```powershell
powershell -ExecutionPolicy Bypass -File tests/offline/windows_release_smoke.ps1 -PackageDir <extracted-package-dir>
```

Deferral is an explicit evidence status, not a substitute for execution and not a fabricated success claim. These native smokes remain Task 17-only evidence and must never be used to block Task 16 acceptance.

## Current status

Task 17 remains formally `pending` because Task 16 is still the first incomplete task. The release-specific evidence is now recorded:

- `task17PackagingContract`: **pass** — `tests/offline/task17_packaging_test.sh` passes.
- `task17ThreePlatformBuildEvidence`: **pass** — darwin-arm64, darwin-x64 and windows-x64 archives built and statically verified (archive, `.sha256`, package staging, checksum/manifest/offline verification). Evidence: `tests/offline/evidence/task17-build-evidence.json` (OpenCode v1.18.11, 3/3 checks).
- `task17MacOSNativeOfflineEvidence`: **deferred** — native macOS offline smoke not yet run.
- `task17WindowsNativeOfflineEvidence`: **deferred** — native Windows offline smoke not yet run.

No native macOS/Windows PASS is claimed. When Task 16 closes, Task 17 can be activated for final acceptance, with the two native smoke items retained as `deferred` until their target environments are available.

## Scope boundary

Task 17 covers first-install/offline release packaging. Task 18 transactional upgrade, migration, rollback, crash recovery and version switching remain untouched and pending.
