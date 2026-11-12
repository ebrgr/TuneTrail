---
id: ISS-059
title: 059 — Adotar Backlog.md para issues, estudos, roadmaps e decisões
status: Aguardando aceite
assignee: []
created_date: '2026-11-12T23:42:00Z'
updated_date: '2026-11-12T23:42:00Z'
labels:
  - processo
  - docs
milestone: m-6
dependencies: []
documentation:
  - decision-007
priority: medium
type: chore
ordinal: 59000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Substituir os registros em Markdown de `docs-dev/` pelo Backlog.md, com quadro, consultas e histórico por tarefa.

Estudo e comparação: kanban-md, gitplan, Aver, gh-issue-sync e Backlog.md (testes em diretórios descartáveis). Decisão: decision-007.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 backlog/ na raiz com config.yml do projeto
- [x] #2 58 issues importadas com IDs ISS-001–ISS-058 e status preservados
- [x] #3 Roadmaps, estudos, design, PRs e índice importados como documentos
- [x] #4 Estudos privados no quadro e fora do Git
- [x] #5 agents.md descreve o novo fluxo
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-12T23:42:00Z

---

Verificação: `backlog doctor` sem IDs duplicados ou ciclos; 488 links relativos válidos; 47 tarefas Done, 1 Em andamento, 5 Aguardando aceite, 5 Abertas; 5 estudos privados ignorados pelo Git.

Ajuste: status final renomeado de "Fechada" para "Done", porque milestones e estatísticas do Backlog.md 1.55.3 só reconhecem Done/Complete/Completed.

---
<!-- COMMENTS:END -->
