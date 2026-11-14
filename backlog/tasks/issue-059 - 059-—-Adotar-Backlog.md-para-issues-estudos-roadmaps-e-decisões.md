---
id: ISSUE-059
title: 059 — Adotar Backlog.md para issues, estudos, roadmaps e decisões
status: Done
assignee: []
created_date: '2026-11-12T23:42:00Z'
updated_date: '2026-11-12T23:59:00Z'
labels:
  - processo
  - docs
milestone: m-6
dependencies: []
documentation:
  - decision-007
priority: medium
type: chore
ordinal: 66000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Substituir os registros em Markdown de `docs-dev/` pelo Backlog.md, com quadro, consultas e histórico por tarefa.

Estudo e comparação: kanban-md, gitplan, Aver, gh-issue-sync e Backlog.md (testes em diretórios descartáveis). Decisão: decision-007.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 backlog/ na raiz com config.yml do projeto
- [x] #2 58 issues importadas com IDs ISSUE-001–ISSUE-058 e status preservados
- [x] #3 Roadmaps, estudos, design, PRs e índice importados como documentos
- [x] #4 Estudos privados no quadro e fora do Git
- [x] #5 agents.md descreve o novo fluxo
- [x] #6 Estudo e milestone renomeados de TuneTrail 2.0 para TuneTrail 1.0
- [x] #7 Agentes em /.agents, fora do Git, e agents.md atualizado
- [x] #8 docs-dev removido; evidências migradas para backlog/evidencias com links e scripts atualizados
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-12T23:42:00Z

---

Verificação: `backlog doctor` sem IDs duplicados ou ciclos; 488 links relativos válidos; 47 tarefas Done, 1 Em andamento, 5 Aguardando aceite, 5 Abertas; 5 estudos privados ignorados pelo Git.

Ajuste: status final renomeado de "Fechada" para "Done", porque milestones e estatísticas do Backlog.md 1.55.3 só reconhecem Done/Complete/Completed.

---

author: claude
created: 2026-11-12T23:57:00Z

---

Refinamento solicitado pelo usuário:
- Estudo doc-010 e milestone m-8 renomeados para **TuneTrail 1.0** (projeto na 0.7; marco de versão reescrito: fases 0–2 em 0.8–0.9, candidatura ao 1.0).
- Agentes movidos para `/.agents/` e excluídos do Git; `agents.md` lista advisor, golang-master, deslop e ui-craft.
- `docs-dev/` removido após verificar que as 88 páginas tinham cópia integral no Backlog.md; 550 arquivos de evidência movidos com `git mv` para `backlog/evidencias/`.

Verificação: 872 links relativos válidos; `go test ./...` sem falhas (27 pacotes); `backlog doctor` limpo; 59 tarefas, 30 documentos, 7 decisões.

---

author: claude
created: 2026-11-12T23:59:00Z

---

Aceite: aprovada pelo usuário em 12/11/2026. Removido também o arquivo obsoleto `agents - cópia.md`.

---
<!-- COMMENTS:END -->
