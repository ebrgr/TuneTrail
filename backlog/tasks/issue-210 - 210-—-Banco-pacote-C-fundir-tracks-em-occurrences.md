---
id: ISSUE-210
title: '210 — Banco, pacote C: fundir tracks em occurrences'
status: Aberta
assignee: []
created_date: '2026-11-24T00:51:00Z'
updated_date: '2026-11-28T01:53:00Z'
labels:
  - banco
  - catalogo
milestone: m-11
dependencies:
  - ISSUE-195
  - ISSUE-224
references:
  - >-
    backlog/docs/estudos/doc-047 -
    Auditoria-de-uso-e-repetição-dos-dados-do-banco-schema-23.md
  - backlog/evidencias/db-auditoria/colunas.csv
priority: low
type: chore
ordinal: 223000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-046 §5.2 e §5.5 (pacote C). tracks é a faixa do Koito v2, mantida como ponte: cada tracks.id é igual ao occurrences.id e igual ao recordings.id no ensaio (2.435 = 2.435 = 2.435), com 7 gatilhos de ponte (trg_identity_occurrence_bridge_update, trg_identity_occurrence_insert, trg_identity_track_delete, trg_identity_track_id_immutable, trg_identity_track_insert, trg_identity_track_recording_mbid, trg_identity_track_update) e a coluna occurrences.legacy_track_id. tracks ainda é lida em 37 pontos e escrita em 8 do Go de produção. Fundir tracks em occurrences, renomear listens.track_id para occurrence_id e remover a ponte; cada conceito passa a ter um nome só. Efeito: uma tabela e 7 gatilhos a menos, 227 bytes por música nas quatro tabelas do catálogo (parte é necessária). RISCO: maior refatoração do banco, com regressão possível em estatísticas, rankings, edição e exportação; por isso depois do upgrade de produção e da ISSUE-195, com Reviewer em cada corte e plano de consulta antes e depois. Depende de decisão do usuário (doc-046 §8, decisão 4).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 tracks, a coluna legacy_track_id e os 7 gatilhos de ponte removidos; listens.occurrence_id no lugar de listens.track_id
- [ ] #2 Estatísticas, rankings, edição, fusão, exportação e reimportação com os mesmos resultados, provados por testes e por comparação de digests
- [ ] #3 Planos de consulta e tempos antes e depois
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-27T15:21:00Z

---

Decisão do usuário em 27/11/2026 (B6): confirmado, só depois da subida da produção (ISSUE-224).

---

author: claude
created: 2026-11-28T01:53:00Z

---

Atenção: os gatilhos da migração 026 (ISSUE-218) usam INDEXED BY com nomes de índices das tabelas que esta tarefa recria; ver ISSUE-237.

---
<!-- COMMENTS:END -->
