---
id: ISSUE-092
title: 092 — Semântica das estatísticas, médias e desempenho em históricos grandes
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-20T22:26:00Z'
labels:
  - api
  - validacao
milestone: m-13
dependencies: []
references:
  - backlog/evidencias/issue-092/SEMANTICA.md
  - backlog/evidencias/issue-092/BENCHMARK.md
documentation:
  - doc-044
priority: high
type: spike
ordinal: 92000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-017 (ISSUE-018) e doc-010 §6. Decidir e documentar a semântica das médias (contagem filtrada × dias ativos históricos) antes de alterar cálculos; medir consultas de intervalo em base grande antes de propor índices.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Semântica aprovada pelo usuário e documentada
- [x] #2 Benchmark com histórico grande registrado
- [ ] #3 Tarefas de correção abertas somente se necessárias
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-20T21:57:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-092` na branch `spike/issue-092-semantica-das-estatisticas-medias-e` criada a partir de `main`.

---

author: codex
created: 2026-11-20T22:26:00Z

---

Verificação da spike na v0.47: doc-044 inventaria as fórmulas e registra as opções. Reproduzimos 60 escutas com médias de 0,50 em /stats e 1,94 em /summary; a fixture autônoma mostrou 1 escuta contabilizada com histórico e top vazios. Benchmark em bases descartáveis de 300 mil e 1 milhão, com planos, percentis, recursos e custo do índice experimental sobre a ingestão legada. Em 1 milhão, p95 da Home em Tudo de 2,607 s para 2,036 s com o índice e +17,85 MB no banco consolidado. AC2 verificado. AC1 depende de aceite da semântica e AC3 da decisão sobre as propostas; as tarefas seguem apenas listadas.

---
<!-- COMMENTS:END -->
