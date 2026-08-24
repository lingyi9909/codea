#!/usr/bin/env python3
import argparse
import json
import pathlib
import subprocess
import sys

MAX_INSTALL_MS = 5 * 60 * 1000


def load(path):
    return json.loads(pathlib.Path(path).read_text())


def require(cond, message):
    if not cond:
        raise SystemExit(message)


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--source-commit", required=True)
    p.add_argument("--mac-evidence", required=True)
    p.add_argument("--mac-duration-ms", required=True, type=int)
    p.add_argument("--windows-evidence", required=True)
    p.add_argument("--windows-duration-ms", required=True, type=int)
    p.add_argument("--out-dir", required=True)
    a = p.parse_args()

    mac = load(a.mac_evidence)
    win = load(a.windows_evidence)
    require(mac.get("platform") in {"darwin-arm64", "darwin-x64"}, "macOS evidence platform invalid")
    require(win.get("platform") == "windows-x64", "Windows evidence platform invalid")
    require(win.get("nativeWindows") is True and win.get("wslUsed") is False, "Windows evidence must be native, no WSL")

    for label, ev in (("macOS", mac), ("Windows", win)):
        require(ev.get("publicHttpsBlocked") is True, f"{label}: public HTTPS was not blocked")
        require(ev.get("installerPassed") is True, f"{label}: installer did not pass")
        require(ev.get("opencodeServeHealthy") is True, f"{label}: OpenCode did not start")
        require(ev.get("codeaLauncherStarted") is True, f"{label}: Codea launcher did not start")
        require(ev.get("externalPackageManagerInvocations") == 0, f"{label}: startup invoked an external package manager")
        require(ev.get("enterprisePluginToolsRegistered") == 8, f"{label}: enterprise plugin tools were not 8/8")

    require(a.mac_duration_ms <= MAX_INSTALL_MS, "macOS G1 smoke exceeded 5 minute limit")
    require(a.windows_duration_ms <= MAX_INSTALL_MS, "Windows G1 smoke exceeded 5 minute limit")

    out = pathlib.Path(a.out_dir)
    out.mkdir(parents=True, exist_ok=True)
    writer = pathlib.Path(__file__).with_name("write-release-gate.py")
    evidence = {
        "G1": f"native macOS + Windows completely-offline install/start smokes; durations={a.mac_duration_ms}ms/{a.windows_duration_ms}ms (<5 minute limit)",
        "G2": "native macOS + Windows offline startup; package-manager sentinel npm/bun/pip/pip3/mvn invocation count=0",
        "G2.1": "native macOS + Windows locked OpenCode v1.18.11 live /experimental/tool/ids exposes 8/8 bundled enterprise plugin tools while offline",
    }
    for gate, detail in evidence.items():
        target = out / (gate.replace(".", "_") + ".json")
        subprocess.run([
            sys.executable, str(writer), "--id", gate, "--source-commit", a.source_commit,
            "--status", "pass", "--evidence", detail, "--out", str(target),
        ], check=True)
    print("native release gates derived: G1/G2/G2.1 PASS")


if __name__ == "__main__":
    main()
