import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[2]
SIGN = ROOT / "packaging/platform/windows/sign-release.ps1"
VERIFY = ROOT / "packaging/platform/windows/verify-signature.ps1"
FINALIZE = ROOT / "packaging/platform/windows/finalize-release.ps1"
RELEASE = ROOT / ".github/workflows/windows-release.yml"
TASK27 = ROOT / ".github/workflows/task27-windows-trust-gates.yml"


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

    def test_signtool_discovery_is_bounded(self):
        text = SIGN.read_text()
        self.assertIn("Windows Kits", text)
        self.assertNotIn("-Recurse", text)
        self.assertIn("x64", text)

    def test_verification_requires_valid_authenticode(self):
        text = VERIFY.read_text()
        self.assertIn("Get-AuthenticodeSignature", text)
        self.assertIn("Valid", text)
        self.assertIn("Thumbprint", text)
        self.assertIn("throw", text)

    def test_ci_proof_keygen_avoids_windows_cng_hang(self):
        text = TASK27.read_text()
        self.assertIn("openssl.exe", text)
        self.assertIn("certutil.exe", text)
        self.assertNotIn("New-SelfSignedCertificate", text)
        self.assertNotIn("[Security.Cryptography.RSA]::Create", text)

    def test_stable_release_requires_signing_credentials_through_finalizer(self):
        workflow = RELEASE.read_text()
        finalizer = FINALIZE.read_text()
        self.assertIn("CODEA_WINDOWS_SIGNING_PFX_BASE64", workflow)
        self.assertIn("CODEA_WINDOWS_SIGNING_PFX_PASSWORD", workflow)
        self.assertIn("finalize-release.ps1", workflow)
        self.assertIn("stable", finalizer)
        self.assertIn("CODEA_WINDOWS_SIGNING_PFX_BASE64", finalizer)
        self.assertIn("CODEA_WINDOWS_SIGNING_PFX_PASSWORD", finalizer)
        self.assertIn("sign-release.ps1", finalizer)
        self.assertIn("verify-signature.ps1", finalizer)
        self.assertIn("throw", finalizer)


if __name__ == "__main__":
    unittest.main()
