---
id: ISSUE-092
title: 092 — Semântica das estatísticas, médias e desempenho em históricos grandes
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-20T22:56:00Z'
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
- [x] #1 Semântica aprovada pelo usuário e documentada
- [x] #2 Benchmark com histórico grande registrado
- [x] #3 Tarefas de correção abertas somente se necessárias
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

author: claude
created: 2026-11-20T22:56:00Z

---

Aceite do usuário em 20/11/2026: semântica aprovada (média sobre dias civis do recorte incluindo zeros; período em andamento limitado ao instante da consulta com a data de hoje no denominador e rótulo «até agora»; ranking por gravação como padrão com alternância por lançamento; eventos aceitos, com ocorrências autônomas no balde «Sem lançamento», e pendências fora dos totais) e tarefas aprovadas, com o ajuste de que o fuso NÃO aparece na interface. Tarefas abertas: ISSUE-176 (alinhar a média diária ao período consultado, bug medido: /stats 0,50 × /summary 1,94 para as mesmas 60 escutas) e ISSUE-177 (reduzir leituras históricas repetidas na Home e no Rewind, depois da ISSUE-135). O índice experimental fica sem adoção até a nova medição com os leitores por ocorrência. A preferência persistente de ranking por perfil segue como decisão independente da ISSUE-135. Estudo: doc-044 e backlog/evidencias/issue-092.

---
<!-- COMMENTS:END -->
