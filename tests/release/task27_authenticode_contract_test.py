import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[2]
SIGN = ROOT / "packaging/platform/windows/sign-release.ps1"
VERIFY = ROOT / "packaging/platform/windows/verify-signature.ps1"
FINALIZE = ROOT / "packaging/platform/windows/finalize-release.ps1"
RELEASE = ROOT / ".github/workflows/windows-release.yml"
TASK27 = ROOT / ".github/workflows/task27-windows-trust-gates.yml"
STABLE_E2E = ROOT / "tests/release/task27-stable-finalized-release-e2e.ps1"


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

    def test_verification_error_messages_are_powershell_parse_safe(self):
        text = VERIFY.read_text()
        self.assertIn("${resolved}:", text)
        self.assertNotIn("$resolved:", text)

    def test_ci_proof_identity_is_generated_off_windows(self):
        text = TASK27.read_text()
        self.assertIn("Generate ephemeral Authenticode proof identity on Ubuntu", text)
        self.assertIn("task27-signing-root.cer", text)
        self.assertIn("task27-signing-proof.pfx", text)
        self.assertIn("task27-signing-proof.cer", text)
        self.assertNotIn("New-SelfSignedCertificate", text)
        self.assertNotIn("[Security.Cryptography.RSA]::Create", text)
        self.assertNotIn("Get-Command openssl.exe", text)

    def test_ci_proof_uses_real_root_to_leaf_chain(self):
        text = TASK27.read_text()
        self.assertIn("basicConstraints = critical,CA:TRUE", text)
        self.assertIn("keyUsage = critical,keyCertSign,cRLSign", text)
        self.assertIn("basicConstraints = critical,CA:FALSE", text)
        self.assertIn("keyUsage = critical,digitalSignature", text)
        self.assertIn("openssl x509 -req", text)
        self.assertIn("-CA \"$work/root.pem\"", text)
        self.assertIn("-certfile \"$work/root.pem\"", text)

    def test_ci_trust_import_installs_private_root_and_publisher_noninteractively(self):
        text = TASK27.read_text()
        self.assertIn("X509Store", text)
        self.assertIn("StoreLocation]::LocalMachine", text)
        self.assertIn("'Root'", text)
        self.assertIn("StoreLocation]::CurrentUser", text)
        self.assertIn("'TrustedPublisher'", text)
        self.assertIn("OpenFlags]::ReadWrite", text)
        self.assertNotIn("certutil.exe -user -addstore", text)
        self.assertNotIn("certutil.exe -user -delstore", text)

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

    def test_task27_executes_full_stable_finalizer_e2e(self):
        workflow = TASK27.read_text()
        self.assertTrue(STABLE_E2E.exists())
        e2e = STABLE_E2E.read_text()
        self.assertIn("Stable Finalized Release E2E", workflow)
        self.assertIn("finalize-release.ps1", workflow)
        self.assertIn("-Channel stable", workflow)
        self.assertIn("CODEA_WINDOWS_SIGNING_PFX_BASE64", workflow)
        self.assertIn("task27-stable-finalized-release-e2e.ps1", workflow)
        self.assertIn("Get-AuthenticodeSignature", e2e)
        self.assertIn("manifest.json", e2e)
        self.assertIn(".sha256", e2e)
        self.assertIn("signatureStatus", e2e)
        self.assertIn("signerThumbprint", e2e)
        self.assertIn("task27-windows-installed-lifecycle.ps1", e2e)

    def test_task27_negative_stable_without_credentials_fails_closed(self):
        workflow = TASK27.read_text()
        self.assertIn("Stable without signing credentials fails closed", workflow)
        self.assertIn("CODEA_WINDOWS_SIGNING_PFX_BASE64", workflow)
        self.assertIn("CODEA_WINDOWS_SIGNING_PFX_PASSWORD", workflow)
        self.assertIn("expected stable finalizer to fail without signing credentials", workflow)


if __name__ == "__main__":
    unittest.main()
