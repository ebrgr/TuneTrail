"""Exercise version changes in isolated repositories, without touching project files."""
import subprocess
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("version.py")
FILES = ["Dockerfile", "deploy/compose.yaml", "deploy/.env.example", "README.md", "DEPLOYMENT.md"]


class VersionTests(unittest.TestCase):
    def setUp(self):
        # Each test uses its own disposable repository and version files.
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / "scripts").mkdir()
        (self.root / "scripts/version.py").write_text(SCRIPT.read_text())
        (self.root / "internal/version").mkdir(parents=True)
        self.git("init", "-q")
        self.git("config", "user.name", "Version Test")
        self.git("config", "user.email", "version-test@example.invalid")

    def git(self, *args):
        # Git history is confined to the temporary test directory.
        return subprocess.run(["git", *args], cwd=self.root, check=True, capture_output=True, text=True)

    def files(self, version):
        # Include an upstream version to detect accidental partial replacements.
        (self.root / "internal/version/VERSION").write_text(version + "\n")
        for name in FILES:
            p = self.root / name
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_text(f"current {version}; historical Koito 0.3.2-SUM-RC3 and v0.3.2\n")

    def run_version(self, *args):
        # Inspect both successful and rejected invocations.
        return subprocess.run(["python3", "scripts/version.py", *args], cwd=self.root, capture_output=True, text=True)

    def commit(self):
        # Make the parent comparison use actual commit history.
        self.git("add", ".")
        self.git("commit", "-qm", "test")

    def test_bump_numeric_and_preserve_history(self):
        for current, expected in [("0.3", "0.4"), ("0.9", "0.10"), ("0.99", "0.100")]:
            with self.subTest(current=current):
                self.files(current)
                result = self.run_version("--bump")
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual((self.root / "internal/version/VERSION").read_text().strip(), expected)
                for name in FILES:
                    text = (self.root / name).read_text()
                    self.assertIn(f"current {expected}", text)
                    self.assertIn("Koito 0.3.2-SUM-RC3 and v0.3.2", text)

    def test_parent_migration_and_numeric_order(self):
        self.files("0.3.2-SUM-RC3")
        self.commit()
        self.files("0.3")
        self.commit()
        self.assertEqual(self.run_version("--check-parent").returncode, 0)
        self.files("0.3")
        self.git("commit", "--allow-empty", "-qm", "same version")
        self.assertNotEqual(self.run_version("--check-parent").returncode, 0)
        self.files("0.4")
        self.commit()
        self.assertEqual(self.run_version("--check-parent").returncode, 0)
        self.files("0.10")
        self.commit()
        self.files("0.9")
        self.commit()
        self.assertNotEqual(self.run_version("--check-parent").returncode, 0)

    def test_not_lower_allows_unchanged_version(self):
        # Ordinary commits keep the version; only a regression is rejected.
        self.files("0.8")
        self.commit()
        self.git("commit", "--allow-empty", "-qm", "same version")
        self.assertEqual(self.run_version("--check-not-lower").returncode, 0)
        self.assertNotEqual(self.run_version("--check-parent").returncode, 0)
        self.files("0.9")
        self.commit()
        self.assertEqual(self.run_version("--check-not-lower").returncode, 0)
        self.files("0.7")
        self.commit()
        result = self.run_version("--check-not-lower")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("decreased", result.stderr)

    def test_legacy_migration_after_bump(self):
        # A first TuneTrail commit may already have been bumped to 0.4.
        self.files("0.3.2-SUM-RC3")
        self.commit()
        self.files("0.4")
        self.commit()
        self.assertEqual(self.run_version("--check-parent").returncode, 0)
        self.files("0.2")
        self.assertNotEqual(self.run_version("--check-parent").returncode, 0)

    def test_invalid_version_rejected(self):
        for version in ["0.03", "0.3.2", "1.0", "0.3-SUM-RC3"]:
            with self.subTest(version=version):
                self.files(version)
                self.assertNotEqual(self.run_version().returncode, 0)


if __name__ == "__main__":
    unittest.main()
