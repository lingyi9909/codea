#!/usr/bin/env python3
import argparse
import json
import pathlib
import sys

GATES = ["G1","G2","G2.1","G3","G4","G5","G6","G7","G8","G9","G10","G11","G12","G12.1","G13","G14","G15"]


def fail(message: str) -> int:
    print(f"release closeout failed: {message}", file=sys.stderr)
    return 1


def load_json(path: pathlib.Path):
    try:
        return json.loads(path.read_text())
    except Exception as exc:
        raise ValueError(f"read {path}: {exc}") from exc


def clean_cell(value: str) -> str:
    return " ".join(value.split()).replace("|", "\\|")


def validate(cert_payload, gates):
    if cert_payload.get("schemaVersion") != 1:
        raise ValueError("unsupported certification schemaVersion")
    if cert_payload.get("passed") is not True:
        raise ValueError("strict certification artifact is not passed")

    certification = cert_payload.get("certification")
    if not isinstance(certification, dict):
        raise ValueError("certification object is missing")
    source_commit = certification.get("sourceCommit")
    if not isinstance(source_commit, str) or not source_commit.strip():
        raise ValueError("certification sourceCommit is missing")

    if not isinstance(gates, list):
        raise ValueError("release gates must be a JSON array")
    ids = [gate.get("id") for gate in gates if isinstance(gate, dict)]
    if ids != GATES:
        raise ValueError(f"release gate order/set mismatch: expected {GATES}, got {ids}")

    for gate in gates:
        gate_id = gate["id"]
        if gate.get("sourceCommit") != source_commit:
            raise ValueError(f"gate {gate_id} sourceCommit does not match certification sourceCommit")
        if gate.get("status") != "pass":
            raise ValueError(f"gate {gate_id} is not pass: {gate.get('status')}")
        evidence = gate.get("evidence")
        if not isinstance(evidence, str) or not evidence.strip():
            raise ValueError(f"gate {gate_id} evidence is empty")

    parity = certification.get("parity")
    if not isinstance(parity, dict):
        raise ValueError("certification parity result is missing")
    total = parity.get("total")
    passed = parity.get("passed")
    if total != 12 or passed != 12:
        raise ValueError(f"required Runtime parity is not 12/12: {passed}/{total}")
    if certification.get("generalCompletionRate") != 1.0:
        raise ValueError("generalCompletionRate is not 1.0")

    timestamp = cert_payload.get("timestamp")
    if not isinstance(timestamp, str) or not timestamp.strip():
        raise ValueError("certification timestamp is missing")
    return source_commit, timestamp


def render_checklist(source_commit: str, timestamp: str, gates) -> str:
    lines = [
        "# Codea V1 Final Release Certification Checklist",
        "",
        f"- Source commit: `{source_commit}`",
        f"- Certification timestamp: `{timestamp}`",
        f"- Release gates: **17/17 PASS**",
        "- Required dual Runtime parity: **12/12 PASS**",
        "- Final verdict: **RELEASE CERTIFIED**",
        "",
        "## G1–G15 evidence",
        "",
        "| Gate | Status | Evidence |",
        "| --- | --- | --- |",
    ]
    for gate in gates:
        lines.append(f"| {gate['id']} | PASS | {clean_cell(gate['evidence'])} |")
    lines.extend([
        "",
        "## Completion checks",
        "",
        "- [x] G1–G15 aggregated for one exact source commit, including G2.1 and G12.1",
        "- [x] G15 uses real approved-company-intranet Maven/npm/PyPI/Go mirror evidence",
        "- [x] Strict release certifier passed",
        "- [x] Real distinct baseline/candidate Runtime parity is 12/12 PASS",
        "- [x] Final checklist and report generated from machine evidence",
        "- [ ] `docs/execution-state.yaml` updated to completed after this artifact is reviewed and committed",
        "",
    ])
    return "\n".join(lines)


def render_report(source_commit: str, timestamp: str, gates) -> str:
    g15 = next(g for g in gates if g["id"] == "G15")
    return "\n".join([
        "# Codea V1 Final Release Certification Report",
        "",
        "## Verdict",
        "",
        "**RELEASE CERTIFIED**",
        "",
        f"Codea V1 source commit `{source_commit}` has passed the strict final Release Certification at `{timestamp}`.",
        "",
        "## Certified scope",
        "",
        "- G1–G15 plus G2.1/G12.1: **17/17 PASS**",
        "- Real dual Runtime Required scenarios: **12/12 PASS**",
        "- General completion rate: **100%**",
        "- All gate artifacts are bound to the exact same source commit",
        "- G15 is backed by real approved-company-intranet mirror evidence",
        "",
        "## G15 evidence",
        "",
        clean_cell(g15["evidence"]),
        "",
        "## Closeout",
        "",
        "This report is generated only after the strict machine certification artifact is PASS and every required release gate is PASS. A deferred or failed G15 cannot generate this final report.",
        "",
    ])


def write_atomic(path: pathlib.Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp = path.with_name(path.name + ".tmp")
    tmp.write_text(content)
    tmp.replace(path)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--certification", required=True)
    parser.add_argument("--gates", required=True)
    parser.add_argument("--checklist-out", required=True)
    parser.add_argument("--report-out", required=True)
    args = parser.parse_args()

    certification_path = pathlib.Path(args.certification)
    gates_path = pathlib.Path(args.gates)
    checklist_out = pathlib.Path(args.checklist_out)
    report_out = pathlib.Path(args.report_out)

    try:
        cert_payload = load_json(certification_path)
        gates = load_json(gates_path)
        source_commit, timestamp = validate(cert_payload, gates)
        checklist = render_checklist(source_commit, timestamp, gates)
        report = render_report(source_commit, timestamp, gates)
        write_atomic(checklist_out, checklist)
        write_atomic(report_out, report)
    except Exception as exc:
        checklist_out.unlink(missing_ok=True)
        report_out.unlink(missing_ok=True)
        return fail(str(exc))

    print(f"release closeout generated: {source_commit} gates=17/17 parity=12/12")
    return 0


if __name__ == "__main__":
    sys.exit(main())
