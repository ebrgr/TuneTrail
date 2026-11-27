---
id: ISSUE-207
title: '207 — Banco: medir planos e decidir os índices de listens e do recibo'
status: Aberta
assignee: []
created_date: '2026-11-24T00:51:00Z'
updated_date: '2026-11-27T15:21:00Z'
labels:
  - banco
  - ingestao
  - validacao
milestone: m-11
dependencies: []
references:
  - >-
    backlog/docs/estudos/doc-047 -
    Auditoria-de-uso-e-repetição-dos-dados-do-banco-schema-23.md
  - backlog/evidencias/db-auditoria/colunas.csv
priority: medium
type: spike
ordinal: 220000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-047 §2 e §5 (nível 3). Em 300 mil escutas migradas do v2, três índices de listens parecem redundantes e somam cerca de 10 MB (9% do banco): idx_listens_profile_listened_at (prefixo de idx_listens_event_cursor), idx_listens_track_id (prefixo de idx_listens_track_id_listened_at) e idx_listens_user_id (user_id é só atribuição e vazia nas escutas novas). Em ingest_events há 20 índices que custam 598 bytes por escuta; os quatro maiores (idx_ingest_source_receipt, idx_ingest_health_decisions, idx_ingest_inbox_example e idx_ingest_health_received) somam 328 bytes por escuta e existem para as páginas de saúde. Para cada índice: EXPLAIN QUERY PLAN das consultas das páginas (saúde, Inbox, fila de revisão, Home, rankings, exportação) antes e depois de removê-lo, com 300 mil escutas, e uma decisão registrada. A ISSUE-190 mostrou que o planejador escolhe idx_listens_track_id hoje, então remover muda planos. Entregar uma tabela índice, uso, custo, decisão. Não altera o esquema por si; a remoção entra no arquivo único da ISSUE-195. Depende de decisão do usuário (doc-047 §7, decisão 4).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Plano de consulta antes e depois, com 300 mil escutas, para cada índice candidato e cada página que o usa
- [ ] #2 Decisão por índice (remover, manter ou trocar) registrada com a economia em MB
- [ ] #3 Lista de índices a remover entregue à ISSUE-195
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-27T15:21:00Z

---

Decisão do usuário em 27/11/2026: medição dos índices autorizada (B3).

---
<!-- COMMENTS:END -->
