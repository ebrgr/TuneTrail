---
id: ISSUE-206
title: >-
  206 — Banco: pequenas simplificações (revisão por perfil, origens e ordem de
  exportação)
status: Aberta
assignee: []
created_date: '2026-11-24T00:51:00Z'
updated_date: '2026-11-27T15:21:00Z'
labels:
  - banco
  - ingestao
milestone: m-11
dependencies: []
references:
  - >-
    backlog/docs/estudos/doc-047 -
    Auditoria-de-uso-e-repetição-dos-dados-do-banco-schema-23.md
  - backlog/evidencias/db-auditoria/colunas.csv
priority: low
type: chore
ordinal: 219000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-046 §5.2 e doc-047 R4. Três simplificações pequenas e independentes. (1) profile_ingest_review (uma linha por perfil só quando o padrão muda, lida como coalesce(..., 1)) vira a coluna listening_profiles.ingest_review INTEGER NOT NULL DEFAULT 1 CHECK(0,1). (2) event_origins (9 valores fixos, nenhuma leitura, só a chave estrangeira e um INSERT de manual) vira CHECK(origin IN (...)) em listens, e o gatilho trg_identity_listen_origin_insert sai. (3) listen_export_order (uma linha por escuta repetindo event_id, 94 bytes por escuta, 4% do banco) vira uma coluna de sequência que nunca se repete, mantendo a exportação retomável da ISSUE-183. Estimativa: 3 tabelas e 1 gatilho a menos e cerca de 86 bytes a menos por escuta. RISCO do item 3: a sequência não pode reutilizar números mesmo se a escuta mais nova for apagada (AUTOINCREMENT); medir antes. Entra no arquivo único da ISSUE-195. Depende de decisão do usuário (doc-046 §8, decisão 3; doc-047 §7).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 listening_profiles guarda a revisão por perfil e profile_ingest_review sai
- [ ] #2 listens.origin protegido por CHECK e event_origins removida
- [ ] #3 Exportação retomável preservada sem listen_export_order, com sequência que nunca se repete e teste de interrupção e retomada
- [ ] #4 Bytes por escuta medidos antes e depois
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-27T15:21:00Z

---

Decisão do usuário em 27/11/2026: pacote B aprovado antes da 195 (A4), com migração própria (A6).

---
<!-- COMMENTS:END -->
