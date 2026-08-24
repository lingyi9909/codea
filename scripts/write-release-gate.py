#!/usr/bin/env python3
import argparse
import json
import pathlib
import sys

GATES = {"G1","G2","G2.1","G3","G4","G5","G6","G7","G8","G9","G10","G11","G12","G12.1","G13","G14","G15"}
STATUSES = {"pass", "fail", "deferred"}


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("--id", required=True)
    p.add_argument("--source-commit", required=True)
    p.add_argument("--status", required=True)
    p.add_argument("--evidence", required=True)
    p.add_argument("--out", required=True)
    a = p.parse_args()
    if a.id not in GATES:
        p.error(f"unknown gate id: {a.id}")
    if a.status not in STATUSES:
        p.error(f"invalid status: {a.status}")
    if not a.source_commit.strip():
        p.error("sourceCommit is required")
    if not a.evidence.strip():
        p.error("evidence is required")
    out = pathlib.Path(a.out)
    out.parent.mkdir(parents=True, exist_ok=True)
    payload = {
        "id": a.id,
        "status": a.status,
        "evidence": a.evidence,
        "sourceCommit": a.source_commit,
    }
    out.write_text(json.dumps(payload, indent=2) + "\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
