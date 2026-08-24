#!/usr/bin/env python3
import json
import pathlib
import subprocess
import sys
import tempfile
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[2]
WRITE = ROOT / "scripts" / "write-release-gate.py"
MERGE = ROOT / "scripts" / "merge-release-gates.py"
GATES = ["G1","G2","G2.1","G3","G4","G5","G6","G7","G8","G9","G10","G11","G12","G12.1","G13","G14","G15"]


class ReleaseGateToolsTest(unittest.TestCase):
    def run_cmd(self, *args, expect=0):
        p = subprocess.run([sys.executable, *map(str, args)], text=True, capture_output=True)
        self.assertEqual(p.returncode, expect, msg=f"stdout={p.stdout}\nstderr={p.stderr}")
        return p

    def test_write_gate_records_source_commit_and_evidence(self):
        with tempfile.TemporaryDirectory() as td:
            out = pathlib.Path(td) / "G3.json"
            self.run_cmd(WRITE, "--id", "G3", "--source-commit", "abc123", "--status", "pass", "--evidence", "go test ./internal/skill", "--out", out)
            payload = json.loads(out.read_text())
            self.assertEqual(payload, {
                "id": "G3",
                "status": "pass",
                "evidence": "go test ./internal/skill",
                "sourceCommit": "abc123",
            })

    def test_merge_requires_exact_gate_set_and_same_source(self):
        with tempfile.TemporaryDirectory() as td:
            d = pathlib.Path(td) / "gates"
            d.mkdir()
            for gate in GATES:
                (d / f"{gate.replace('.', '_')}.json").write_text(json.dumps({
                    "id": gate, "status": "pass", "evidence": f"evidence:{gate}", "sourceCommit": "abc123"
                }))
            out = pathlib.Path(td) / "release-gates.json"
            self.run_cmd(MERGE, "--source-commit", "abc123", "--input-dir", d, "--out", out)
            payload = json.loads(out.read_text())
            self.assertEqual([x["id"] for x in payload], GATES)
            self.assertTrue(all(x["sourceCommit"] == "abc123" for x in payload))

    def test_merge_rejects_missing_or_stale_gate(self):
        with tempfile.TemporaryDirectory() as td:
            d = pathlib.Path(td) / "gates"
            d.mkdir()
            for gate in GATES[:-1]:
                (d / f"{gate.replace('.', '_')}.json").write_text(json.dumps({
                    "id": gate, "status": "pass", "evidence": gate, "sourceCommit": "abc123"
                }))
            out = pathlib.Path(td) / "release-gates.json"
            p = subprocess.run([sys.executable, str(MERGE), "--source-commit", "abc123", "--input-dir", str(d), "--out", str(out)], text=True, capture_output=True)
            self.assertNotEqual(p.returncode, 0)
            self.assertIn("missing", (p.stdout + p.stderr).lower())

            (d / "G15.json").write_text(json.dumps({
                "id": "G15", "status": "pass", "evidence": "mirror", "sourceCommit": "old456"
            }))
            p = subprocess.run([sys.executable, str(MERGE), "--source-commit", "abc123", "--input-dir", str(d), "--out", str(out)], text=True, capture_output=True)
            self.assertNotEqual(p.returncode, 0)
            self.assertIn("sourcecommit", (p.stdout + p.stderr).lower())


if __name__ == "__main__":
    unittest.main()
