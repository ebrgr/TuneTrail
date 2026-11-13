#!/usr/bin/env python3
"""Drive the Backlog.md issue cycle: create, start, send for review and publish.

Publishing an accepted task (status Done with an "Aceite" comment) rebases its
branch onto main, runs the checks, bumps the 0.x version, rolls the CHANGELOG,
fast-forwards main and tags the release. Nothing is pushed without --push.
"""
import argparse
from datetime import date
import os
from pathlib import Path
import re
import shlex
import subprocess
import sys
import unicodedata

ROOT = Path(__file__).resolve().parent.parent
TASKS = ROOT / "backlog/tasks"
VERSION_FILE = ROOT / "internal/version/VERSION"
CHANGELOG = ROOT / "CHANGELOG.md"
# Overridable so tests can run in a disposable repository without real tools.
BACKLOG = shlex.split(os.environ.get("BACKLOG_BIN", "backlog"))
CHECK_CMD = os.environ.get("CHECK_CMD", "make check")
MAIN = os.environ.get("MAIN_BRANCH", "main")
REMOTE = os.environ.get("PUSH_REMOTE", "origin")


def read_prefix():
    # Task IDs follow Backlog.md's task_prefix (e.g. "issue" -> ISSUE-060).
    config = ROOT / "backlog/config.yml"
    match = re.search(r'^task_prefix:\s*["\']?([A-Za-z]+)', config.read_text(), re.M) if config.exists() else None
    return (match[1] if match else "task").lower()


PREFIX = read_prefix()
BRANCH_PREFIX = {"bug": "fix", "feature": "feat", "enhancement": "feat", "docs": "docs", "spike": "spike"}


class Fail(Exception):
    pass


def run(*cmd, capture=True, check=True):
    # Every external command runs from the repository root; failures stop the flow.
    result = subprocess.run(cmd, cwd=ROOT, capture_output=capture, text=True)
    if check and result.returncode != 0:
        detail = (result.stderr or result.stdout or "").strip() if capture else ""
        raise Fail(f"falhou: {' '.join(cmd)}" + (f"\n{detail}" if detail else ""))
    return result.stdout.strip() if capture else ""


def git(*args, **kw):
    return run("git", *args, **kw)


def backlog(*args):
    return run(*BACKLOG, *args, "--plain")


def task_number(value):
    # Accept 60, 060, ISSUE-060 or issue-60 and normalize to the zero-padded number.
    match = re.fullmatch(rf"(?i)(?:{PREFIX}-)?0*(\d+)", value.strip())
    if not match:
        raise Fail(f"tarefa inválida: {value}")
    return f"{int(match[1]):03d}"


def task_id(num):
    return f"{PREFIX.upper()}-{num}"


def task_file(num):
    matches = sorted(TASKS.glob(f"{PREFIX}-{num} - *.md"))
    if len(matches) != 1:
        raise Fail(f"{task_id(num)}: esperado um arquivo em backlog/tasks, encontrados {len(matches)}")
    return matches[0]


def read_task(num):
    # Parse only the fields the workflow needs from the Backlog.md frontmatter.
    path = task_file(num)
    text = path.read_text()
    front = text.split("---", 2)[1] if text.startswith("---") else ""
    fields = {}
    for key in ("title", "status", "type"):
        match = re.search(rf"^{key}:\s*(.*)$", front, re.M)
        value = match[1].strip() if match else ""
        if len(value) >= 2 and value[0] == value[-1] and value[0] in "'\"":
            value = value[1:-1].replace("''", "'")
        fields[key] = value
    comments = text.split("<!-- COMMENTS:BEGIN -->", 1)[-1] if "<!-- COMMENTS:BEGIN -->" in text else ""
    fields["accepted"] = bool(re.search(r"^Aceite\b", comments, re.M | re.I))
    fields["path"] = path
    return fields


def plain_title(title):
    # Drop the "NNN — " prefix used by task titles.
    return re.sub(r"^\d+\s*[—-]\s*", "", title)


def slugify(text, words=5):
    ascii_text = unicodedata.normalize("NFKD", text).encode("ascii", "ignore").decode()
    parts = re.findall(r"[a-z0-9]+", ascii_text.lower())
    return "-".join(parts[:words]) or "tarefa"


def branch_name(num, task):
    prefix = BRANCH_PREFIX.get(task["type"].lower(), "chore")
    return f"{prefix}/{PREFIX}-{num}-{slugify(plain_title(task['title']))}"


def find_branch(num):
    branches = git("for-each-ref", "refs/heads", "--format=%(refname:short)").splitlines()
    found = [b for b in branches if re.fullmatch(rf"[^/]+/{PREFIX}-{num}(-.*)?", b)]
    if len(found) > 1:
        raise Fail(f"{task_id(num)}: mais de uma branch: {', '.join(found)}")
    return found[0] if found else None


def dirty_paths(num):
    # Tracked changes, except the task's own record, which the flow commits itself.
    paths = git("-c", "core.quotepath=false", "diff", "--name-only", "HEAD").splitlines()
    return [p for p in paths if not p.startswith(f"backlog/tasks/{PREFIX}-{num} - ")]


def require_clean(num):
    dirty = dirty_paths(num)
    if dirty:
        raise Fail("árvore com mudanças não commitadas:\n  " + "\n  ".join(dirty))


def author():
    return os.environ.get("ISSUE_AUTHOR") or git("config", "user.name") or "agente"


def cmd_new(args):
    # IDs come from Backlog.md; the "NNN — " title prefix must match them.
    ids = [int(m[1]) for p in (ROOT / "backlog").rglob(f"{PREFIX}-*.md") if (m := re.match(rf"{PREFIX}-(\d+)", p.name))]
    guess = f"{max(ids, default=0) + 1:03d}"
    cmd = ["task", "create", f"{guess} — {args.title}", "-s", "Aberta", "--type", args.type]
    if args.labels:
        cmd += ["-l", args.labels]
    if args.description:
        cmd += ["-d", args.description]
    if args.milestone:
        cmd += ["-m", args.milestone]
    out = backlog(*cmd)
    match = re.search(rf"Task {PREFIX}-(\d+)", out, re.I)
    if not match:
        raise Fail("não foi possível ler o ID criado:\n" + out)
    num = f"{int(match[1]):03d}"
    if num != guess:
        backlog("task", "edit", task_id(num), "-t", f"{num} — {args.title}")
    print(f"{task_id(num)} criada: {num} — {args.title}")
    print(f"Próximo passo: make issue.start ISSUE={int(num)}")


def cmd_start(args):
    num = task_number(args.iss)
    task = read_task(num)
    if task["status"] == "Done":
        raise Fail(f"{task_id(num)} já está Done; reabra-a antes de retomar o trabalho")
    require_clean(num)
    branch = find_branch(num)
    if branch:
        git("checkout", branch)
        note = f"Retomada na branch `{branch}`."
    else:
        branch = branch_name(num, task)
        git("checkout", "-b", branch, MAIN)
        note = f"Início: branch `{branch}` criada a partir de `{MAIN}`."
    if task["status"] != "Em andamento":
        backlog("task", "edit", task_id(num), "-s", "Em andamento", "-a", f"@{author()}",
                "--comment", note, "--comment-author", author())
    print(f"{task_id(num)} em andamento na branch {branch}")
    print(f"Commits devem começar com '{task_id(num)}: '")


def cmd_review(args):
    num = task_number(args.iss)
    task = read_task(num)
    branch = find_branch(num)
    current = git("rev-parse", "--abbrev-ref", "HEAD")
    if branch != current:
        raise Fail(f"rode na branch da tarefa ({branch or 'inexistente'}); atual: {current}")
    if task["status"] not in ("Em andamento", "Planejada"):
        raise Fail(f"{task_id(num)} está '{task['status']}'; esperado Em andamento")
    print(f"$ {CHECK_CMD}")
    run(*shlex.split(CHECK_CMD), capture=False)
    backlog("task", "edit", task_id(num), "-s", "Aguardando aceite",
            "--comment", args.message, "--comment-author", author())
    # Agents must relay this notice: the acceptance step is optional but never silent.
    print(f"{task_id(num)} aguardando aceite do usuário")
    print(f"AVISO ao usuário: {task_id(num)} — {plain_title(task['title'])} está em Aguardando aceite.")
    print(f"  Verificação: {args.message}")
    print(f"  Para publicar: registrar o aceite (Done) e rodar make issue.publish ISSUE={int(num)}")


def roll_changelog(text, version, today, ref):
    # Keep a fresh [Unreleased] and turn its current entries into the release section.
    match = re.search(r"^## \[Unreleased\][^\n]*\n?", text, re.M)
    if not match:
        raise Fail("CHANGELOG.md sem a seção '## [Unreleased]'")
    head, rest = text[:match.start()], text[match.end():]
    nxt = re.search(r"^## \[", rest, re.M)
    body, tail = (rest[:nxt.start()], rest[nxt.start():]) if nxt else (rest, "")
    if not body.strip():
        body = f"\n- No user-visible changes ({ref}).\n\n"
    return f"{head}## [Unreleased]\n\n## [{version}] - {today}\n{body}{tail}"


def next_version():
    current = VERSION_FILE.read_text().strip()
    match = re.fullmatch(r"0\.(\d+)", current)
    if not match:
        raise Fail(f"versão inválida: {current}")
    return current, f"0.{int(match[1]) + 1}"


def cmd_publish(args):
    num = task_number(args.iss)
    ref = task_id(num)
    task = read_task(num)
    # Validate everything before touching the repository.
    if task["status"] != "Done":
        raise Fail(f"{ref} está '{task['status']}'; só publica tarefa Done (aceita pelo usuário)")
    if not task["accepted"]:
        raise Fail(f"{ref} está Done sem comentário 'Aceite: ...' do usuário")
    require_clean(num)
    branch = find_branch(num)
    if not branch:
        raise Fail(f"{ref}: nenhuma branch local '*/{PREFIX}-{num}-*'")
    commits = git("log", "--format=%h %s", f"{MAIN}..{branch}").splitlines()
    if not commits:
        raise Fail(f"{branch} não tem commits além de {MAIN}")
    bad = [c for c in commits if not c.split(" ", 1)[1].startswith(f"{ref}: ")]
    if bad:
        raise Fail(f"commits sem o prefixo '{ref}: ':\n  " + "\n  ".join(bad))
    current, version = next_version()
    tag = f"v{version}"
    if git("tag", "--list", tag):
        raise Fail(f"a tag {tag} já existe")
    pending = git("status", "--porcelain", "--", f":(glob)backlog/tasks/{PREFIX}-{num} - *")

    print(f"{ref} — {plain_title(task['title'])}")
    print(f"  branch {branch}: {len(commits)} commit(s) sobre {MAIN}")
    for c in commits:
        print(f"    {c}")
    if pending:
        print(f"  registro da tarefa pendente será commitado como '{ref}: Registra aceite'")
    print(f"  {CHECK_CMD} · versão {current} → {version} · CHANGELOG [{version}] · tag {tag}")
    print(f"  merge --ff-only em {MAIN}" + (f" · push para {REMOTE}" if args.push else " · sem push"))
    if args.push:
        # A pushed v* tag triggers the Docker workflow, which publishes a public image.
        print(f"  ATENÇÃO: o push da tag {tag} dispara .github/workflows/docker.yml e publica a imagem no GHCR (inclusive como latest)")
    if args.dry_run:
        print("DRY: nada foi alterado")
        return

    git("checkout", branch)
    if pending:
        git("add", "-A", "--", f":(glob)backlog/tasks/{PREFIX}-{num} - *")
        git("commit", "-m", f"{ref}: Registra aceite")
    try:
        git("rebase", MAIN)
    except Fail as exc:
        # Leave the branch as it was so the conflict can be resolved by hand.
        git("rebase", "--abort", check=False)
        raise Fail(f"rebase de {branch} sobre {MAIN} falhou e foi desfeito; resolva manualmente\n{exc}")
    print(f"$ {CHECK_CMD}")
    run(*shlex.split(CHECK_CMD), capture=False)

    run(sys.executable, "scripts/version.py", "--bump")
    CHANGELOG.write_text(roll_changelog(CHANGELOG.read_text(), version, date.today().isoformat(), ref))
    backlog("task", "edit", ref, "--comment",
            f"Publicação: versão {version} (tag `{tag}`), branch `{branch}` integrada ao `{MAIN}` por fast-forward.",
            "--comment-author", author())
    git("add", "-u")
    git("add", "-A", "--", f":(glob)backlog/tasks/{PREFIX}-{num} - *")
    git("commit", "-m", f"{ref}: Publica versão {version}\n\n{plain_title(task['title'])}")
    run(sys.executable, "scripts/version.py", "--check-parent")

    git("checkout", MAIN)
    git("merge", "--ff-only", branch)
    git("tag", "-a", tag, "-m", f"{ref} — {plain_title(task['title'])}")
    git("branch", "-d", branch)
    print(f"{ref} publicada localmente: {MAIN} em {git('rev-parse', '--short', 'HEAD')}, tag {tag}")
    if args.push:
        git("push", REMOTE, MAIN, capture=False)
        git("push", REMOTE, tag, capture=False)
        print(f"Enviado para {REMOTE}: {MAIN} e {tag}")
    else:
        print(f"Para enviar (com autorização): git push {REMOTE} {MAIN} {tag}")


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="cmd", required=True)
    p = sub.add_parser("new", help="criar tarefa NNN — Título")
    p.add_argument("title")
    p.add_argument("--type", default="bug", choices=["bug", "feature", "enhancement", "task", "chore", "docs", "spike"])
    p.add_argument("--labels", default="")
    p.add_argument("--description", default="")
    p.add_argument("--milestone", default="")
    p.set_defaults(func=cmd_new)
    p = sub.add_parser("start", help="criar/abrir a branch e marcar Em andamento")
    p.add_argument("iss")
    p.set_defaults(func=cmd_start)
    p = sub.add_parser("review", help="rodar checks e marcar Aguardando aceite")
    p.add_argument("iss")
    p.add_argument("-m", "--message", required=True, help="comentário de verificação")
    p.set_defaults(func=cmd_review)
    p = sub.add_parser("publish", help="publicar tarefa Done com nova versão")
    p.add_argument("iss")
    p.add_argument("--dry-run", action="store_true")
    p.add_argument("--push", action="store_true", help="enviar main e tag (requer autorização do usuário)")
    p.set_defaults(func=cmd_publish)
    args = parser.parse_args()
    try:
        args.func(args)
    except Fail as exc:
        sys.exit(f"issue: {exc}")


if __name__ == "__main__":
    main()
