# Task 17 Report — Offline Distribution Package and Installers

## Overview

Implementation checkpoint: `c9796e2600129dfb0ce476f6192222c403964d48`

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
  - verifies the recorded SHA256 before extraction
  - macOS Bash-compatible metadata parsing
- `packaging/scripts/build-plugins.sh`
  - executes `bun test` + `bun run build`
  - emits only the self-contained ESM bundle
  - rejects bare external imports, including side-effect imports
- `packaging/scripts/collect-skills.sh`
  - packages Code Reviewer, Unit Test and API Documentation Skills
- `packaging/scripts/build-release.sh`
  - closes the full assembly loop: TUI cross-build → runtime → plugin → skills/agents/config → installers → manifest/checksum/offline checks → archive + `.sha256`
- `packaging/scripts/generate-manifest.sh`
  - SHA256 + size for every packaged file
- `packaging/scripts/verify-checksum.sh`
  - verifies manifest schema, safe paths, hash/size, missing files and unmanifested extra files
- `packaging/scripts/verify-offline.sh`
  - rejects plugin package-manager metadata and external package imports
  - checks bundle build-path leakage
  - chains package integrity verification
- `packaging/platform/macos/install.sh`
  - self-contained package-side verification
  - installs under `~/.codea/versions/<version>` (or `CODEA_HOME`)
  - updates `current` and launcher symlinks
- `packaging/platform/windows/install.ps1`
  - verifies manifest hash/size and rejects unmanifested files
  - performs equivalent plugin dependency/import checks
  - installs under `%USERPROFILE%\.codea\versions\<version>` (or `CODEA_HOME`)
  - writes BOM-free `current.txt`, creates a `current` Junction, and a launcher shim without requiring symlink developer privileges
- `tests/offline/task17_packaging_test.sh`
  - positive manifest/integrity/offline checks
  - negative tamper, unmanifested-file, package metadata, external import and side-effect import cases
- `tests/offline/no_public_network_test.sh`
  - fail-closed release-environment gate: public HTTPS must be blocked, packaged runtime must be OpenCode 1.18.11, and static offline checks must pass

## Verification performed in current environment

An isolated synthetic packaging mirror was executed during implementation and passed the initial manifest/checksum/offline contract. After that run, the production scripts were further hardened for macOS Bash compatibility, side-effect imports, self-contained installers, Windows package integrity/current pointer handling, archive creation, and additional negative tests. Those later changes have not been executed as a fresh complete repository Gate in the current sandbox.

## Verification still required before Task 17 acceptance

Run from a repository/build environment with Go 1.26.5, Bun, Python 3, archive tools, and access to the locked OpenCode release asset:

```bash
bash tests/offline/task17_packaging_test.sh

packaging/scripts/build-release.sh darwin-arm64 <out>
packaging/scripts/build-release.sh darwin-x64 <out>
packaging/scripts/build-release.sh windows-x64 <out>
```

Then in an actually network-isolated macOS release test environment, execute the package installer and:

```bash
tests/offline/no_public_network_test.sh <extracted-package-dir>
```

Windows x64 must receive an equivalent real install/runtime smoke using `install.ps1`, confirming install path/current Junction/launcher and packaged `codea.exe` + `opencode.exe` execution without public network access.

Full regression Gate must also include existing Go, Plugin and OpenCode parity suites. Until those fresh results exist, this report does not claim Task 17 Gate PASS.

## Scope boundary

Task 17 only creates first-install/offline release packaging. It does not implement Task 18 transactional upgrade, migration, rollback, crash recovery or version switching logic.
