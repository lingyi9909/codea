#!/usr/bin/env python3
import argparse
import json
import pathlib
import sys

GATES = ["G1","G2","G2.1","G3","G4","G5","G6","G7","G8","G9","G10","G11","G12","G12.1","G13","G14","G15"]


def fail(msg: str) -> int:
    print(f"release gate merge failed: {msg}", file=sys.stderr)
    return 1


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("--source-commit", required=True)
    p.add_argument("--input-dir", required=True)
    p.add_argument("--out", required=True)
    a = p.parse_args()
    root = pathlib.Path(a.input_dir)
    if not root.is_dir():
        return fail(f"input directory missing: {root}")

    by_id = {}
    for path in sorted(root.glob("*.json")):
        try:
            payload = json.loads(path.read_text())
        except Exception as exc:
            return fail(f"parse {path}: {exc}")
        gate_id = payload.get("id")
        if gate_id not in GATES:
            return fail(f"unknown gate {gate_id!r} in {path}")
        if gate_id in by_id:
            return fail(f"duplicate gate {gate_id}")
        if payload.get("sourceCommit") != a.source_commit:
            return fail(f"gate {gate_id} sourceCommit {payload.get('sourceCommit')!r} does not match {a.source_commit!r}")
        if payload.get("status") not in {"pass", "fail", "deferred"}:
            return fail(f"gate {gate_id} has invalid status {payload.get('status')!r}")
        if not isinstance(payload.get("evidence"), str) or not payload["evidence"].strip():
            return fail(f"gate {gate_id} evidence is empty")
        by_id[gate_id] = {
            "id": gate_id,
            "status": payload["status"],
            "evidence": payload["evidence"],
            "sourceCommit": payload["sourceCommit"],
        }

    missing = [gate for gate in GATES if gate not in by_id]
    if missing:
        return fail("missing required gate(s): " + ", ".join(missing))

    out = pathlib.Path(a.out)
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(json.dumps([by_id[gate] for gate in GATES], indent=2) + "\n")
    print(f"release gates merged: {len(GATES)}/{len(GATES)}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
