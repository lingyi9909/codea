import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "packaging/platform/windows/build-security-evidence.ps1"
DOC = ROOT / "docs/release/windows-security-submission.md"


class Task27SecurityEvidenceContract(unittest.TestCase):
    def test_evidence_generator_contains_required_release_identity(self):
        text = SCRIPT.read_text()
        for token in (
            "schemaVersion",
            "codeaVersion",
            "releaseFile",
            "releaseSize",
            "releaseSha256",
            "gitCommit",
            "releaseTag",
            "openCodeVersion",
            "openCodeChecksum",
            "signatureStatus",
            "signerSubject",
            "signerThumbprint",
            "Get-AuthenticodeSignature",
            "ConvertTo-Json",
        ):
            self.assertIn(token, text)
        self.assertNotIn("PfxPassword", text)
        self.assertNotIn("PRIVATE KEY", text)

    def test_submission_doc_does_not_claim_external_acceptance(self):
        text = DOC.read_text()
        self.assertIn("360", text)
        self.assertIn("external", text.lower())
        self.assertIn("不得", text)
        self.assertIn("releaseSha256", text)
        self.assertIn("signerThumbprint", text)


if __name__ == "__main__":
    unittest.main()
