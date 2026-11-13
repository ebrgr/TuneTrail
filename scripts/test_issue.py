"""Exercise the issue workflow in disposable repositories, without touching project files."""
import importlib.util
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).parent
FILES = ["Dockerfile", "deploy/compose.yaml", "deploy/.env.example", "README.md", "DEPLOYMENT.md"]
# The fake Backlog.md appends comments and status changes straight to the task file.
FAKE_BACKLOG = r'''#!/usr/bin/env python3
import sys, glob, re
args = sys.argv[1:]
num = re.sub(r"\D", "", args[2]).zfill(3)
path = glob.glob(f"backlog/tasks/*-{num} - *.md")[0]
text = open(path).read()
if "-s" in args:
    text = re.sub(r"^status: .*$", "status: " + args[args.index("-s") + 1], text, count=1, flags=re.M)
if "--comment" in args:
    text = text.replace("<!-- COMMENTS:END -->", args[args.index("--comment") + 1] + "\n<!-- COMMENTS:END -->")
open(path, "w").write(text)
'''

spec = importlib.util.spec_from_file_location("issue", HERE / "issue.py")
issue = importlib.util.module_from_spec(spec)
spec.loader.exec_module(issue)


class ChangelogTests(unittest.TestCase):
    def test_rolls_unreleased_into_version(self):
        text = "# Changelog\n\n## [Unreleased]\n\n### Added\n\n- New thing.\n\n## [0.7] - 2026-01-01\n\n- Old.\n"
        out = issue.roll_changelog(text, "0.8", "2026-11-13", "ISSUE-060")
        self.assertEqual(out, "# Changelog\n\n## [Unreleased]\n\n## [0.8] - 2026-11-13\n\n### Added\n\n- New thing.\n\n## [0.7] - 2026-01-01\n\n- Old.\n")

    def test_empty_unreleased_records_internal_release(self):
        out = issue.roll_changelog("# Changelog\n\n## [Unreleased]\n", "0.8", "2026-11-13", "ISSUE-060")
        self.assertIn("## [0.8] - 2026-11-13\n\n- No user-visible changes (ISSUE-060).", out)

    def test_branch_names(self):
        self.assertEqual(issue.task_number("ISSUE-060"), "060")
        self.assertEqual(issue.task_number("7"), "007")
        task = {"title": "060 — Corrigir paginação do histórico de escutas antigas", "type": "bug"}
        self.assertEqual(issue.branch_name("060", task), "fix/issue-060-corrigir-paginacao-do-historico-de")


class PublishTests(unittest.TestCase):
    prefix = "issue"

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        for name in ("issue.py", "version.py"):
            (self.root / "scripts").mkdir(exist_ok=True)
            (self.root / "scripts" / name).write_text((HERE / name).read_text())
        fake = self.root.parent / f"{self.root.name}-backlog"
        fake.write_text(FAKE_BACKLOG)
        fake.chmod(0o755)
        self.addCleanup(fake.unlink)
        self.env = {**os.environ, "BACKLOG_BIN": f"{sys.executable} {fake}", "CHECK_CMD": "true", "ISSUE_AUTHOR": "test"}
        self.git("init", "-q", "-b", "main")
        self.git("config", "user.name", "Issue Test")
        self.git("config", "user.email", "issue-test@example.invalid")
        (self.root / "internal/version").mkdir(parents=True)
        (self.root / "internal/version/VERSION").write_text("0.7\n")
        for name in FILES:
            p = self.root / name
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_text("current 0.7\n")
        (self.root / "CHANGELOG.md").write_text("# Changelog\n\n## [Unreleased]\n\n- Visible fix (ISSUE-001).\n")
        (self.root / "backlog").mkdir()
        (self.root / "backlog/config.yml").write_text(f'task_prefix: "{self.prefix}"\n')
        self.task("Done", "Aceite: aprovada.")
        self.git("add", ".")
        self.git("commit", "-qm", "base")
        self.git("checkout", "-qb", f"fix/{self.prefix}-001-teste")
        (self.root / "feature.txt").write_text("x\n")
        self.git("add", ".")
        self.git("commit", "-qm", f"{self.ref}: Corrige teste")

    @property
    def ref(self):
        return f"{self.prefix.upper()}-001"

    def task(self, status, comment):
        tasks = self.root / "backlog/tasks"
        tasks.mkdir(parents=True, exist_ok=True)
        (tasks / f"{self.prefix}-001 - 001-—-Teste.md").write_text(
            f"---\nid: {self.ref}\ntitle: 001 — Teste\nstatus: {status}\ntype: bug\n---\n\n"
            f"<!-- COMMENTS:BEGIN -->\n{comment}\n<!-- COMMENTS:END -->\n")

    def git(self, *args):
        return subprocess.run(["git", *args], cwd=self.root, check=True, capture_output=True, text=True).stdout.strip()

    def publish(self, *args):
        return subprocess.run([sys.executable, "scripts/issue.py", "publish", "1", *args],
                              cwd=self.root, env=self.env, capture_output=True, text=True)

    def test_publish_bumps_tags_and_fast_forwards(self):
        result = self.publish()
        self.assertEqual(result.returncode, 0, result.stderr + result.stdout)
        self.assertEqual(self.git("rev-parse", "--abbrev-ref", "HEAD"), "main")
        self.assertEqual(self.git("log", "-1", "--format=%s"), f"{self.ref}: Publica versão 0.8")
        self.assertEqual(self.git("describe", "--tags"), "v0.8")
        self.assertEqual((self.root / "internal/version/VERSION").read_text(), "0.8\n")
        self.assertIn("## [0.8] - ", (self.root / "CHANGELOG.md").read_text())
        self.assertIn("Publicação: versão 0.8", next((self.root / "backlog/tasks").iterdir()).read_text())
        self.assertEqual(self.git("branch", "--list", f"fix/{self.prefix}-001-teste"), "")
        self.assertEqual(self.git("status", "--porcelain"), "")

    def test_commits_pending_task_record_before_publishing(self):
        self.git("checkout", "-q", "main")
        self.task("Done", "Aceite: aprovada depois.")
        result = self.publish()
        self.assertEqual(result.returncode, 0, result.stderr + result.stdout)
        subjects = self.git("log", "--format=%s", "-3").splitlines()
        self.assertEqual(subjects[:2], [f"{self.ref}: Publica versão 0.8", f"{self.ref}: Registra aceite"])

    def test_dry_run_changes_nothing(self):
        head = self.git("rev-parse", "HEAD")
        result = self.publish("--dry-run")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("0.7 → 0.8", result.stdout)
        self.assertEqual(self.git("rev-parse", "HEAD"), head)
        self.assertEqual(self.git("tag"), "")

    def test_dry_run_push_warns_about_image_publication(self):
        # Dry runs never push; they only announce what PUSH=1 would trigger.
        result = self.publish("--dry-run", "--push")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("publica a imagem no GHCR", result.stdout)
        self.assertEqual(self.git("tag"), "")

    def assertRejected(self, needle):
        head = self.git("rev-parse", "HEAD")
        result = self.publish()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn(needle, result.stderr)
        self.assertEqual(self.git("rev-parse", "HEAD"), head)

    def test_rejects_task_not_done(self):
        self.task("Aguardando aceite", "Verificação: ok.")
        self.assertRejected("só publica tarefa Done")

    def test_rejects_done_without_acceptance(self):
        self.task("Done", "Verificação: ok.")
        self.assertRejected("sem comentário 'Aceite")

    def test_rejects_commit_without_prefix(self):
        (self.root / "other.txt").write_text("y\n")
        self.git("add", ".")
        self.git("commit", "-qm", "wip")
        self.assertRejected(f"sem o prefixo '{self.ref}: '")

    def test_rejects_unrelated_changes(self):
        (self.root / "README.md").write_text("changed\n")
        self.assertRejected("README.md")

    def test_rejects_existing_tag(self):
        self.git("tag", "v0.8", "main")
        self.assertRejected("a tag v0.8 já existe")


class OtherPrefixPublishTests(PublishTests):
    # Same journeys with a different Backlog.md task_prefix, proving it is read from config.
    prefix = "task"


if __name__ == "__main__":
    unittest.main()
