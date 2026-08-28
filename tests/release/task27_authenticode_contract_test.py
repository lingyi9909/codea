import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[2]
SIGN = ROOT / "packaging/platform/windows/sign-release.ps1"
VERIFY = ROOT / "packaging/platform/windows/verify-signature.ps1"
RELEASE = ROOT / ".github/workflows/windows-release.yml"


class Task27AuthenticodeContract(unittest.TestCase):
    def test_signing_script_is_fail_closed_and_sha256(self):
        text = SIGN.read_text()
        self.assertIn("signtool.exe", text)
        self.assertIn("/fd", text)
        self.assertIn("SHA256", text)
        self.assertIn("/tr", text)
        self.assertIn("/td", text)
        self.assertIn("1.3.6.1.5.5.7.3.3", text)
        self.assertIn("finally", text)
        self.assertNotIn("BEGIN PRIVATE KEY", text)

    def test_verification_requires_valid_authenticode(self):
        text = VERIFY.read_text()
        self.assertIn("Get-AuthenticodeSignature", text)
        self.assertIn("Valid", text)
        self.assertIn("Thumbprint", text)
        self.assertIn("throw", text)

    def test_stable_release_requires_signing_credentials(self):
        text = RELEASE.read_text()
        self.assertIn("CODEA_WINDOWS_SIGNING_PFX_BASE64", text)
        self.assertIn("CODEA_WINDOWS_SIGNING_PFX_PASSWORD", text)
        self.assertIn("stable", text)
        self.assertIn("sign-release.ps1", text)
        self.assertIn("verify-signature.ps1", text)


if __name__ == "__main__":
    unittest.main()
