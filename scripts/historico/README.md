# Scripts históricos

Scripts das issues 017–041, mantidos só para reproduzir as evidências registradas em `backlog/evidencias/`. Não rodam contra a versão atual: a maioria compara a interface Go com o React (removido na ISSUE-030) ou exige um build legado.

- Não fazem parte de `make check` nem do CI. A suíte vigente é `scripts/e2e.py` (`make e2e`).
- Os `.py` importam módulos de `scripts/`; para tentar rodá-los, use `PYTHONPATH=scripts`.
- Os `.cjs` exigem Playwright e, em geral, um servidor React histórico.

Inventário e motivo de cada arquivo: `backlog/evidencias/e2e/ISSUE_063.md`.
