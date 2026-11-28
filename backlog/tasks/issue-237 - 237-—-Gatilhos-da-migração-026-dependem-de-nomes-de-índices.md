---
id: ISSUE-237
title: 237 — Gatilhos da migração 026 dependem de nomes de índices
status: Aberta
assignee: []
created_date: '2026-11-28T01:53:00Z'
updated_date: '2026-11-28T01:53:00Z'
labels:
  - banco
milestone: m-11
dependencies:
  - ISSUE-218
type: chore
ordinal: 250000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: parecer do Advisor da ISSUE-218 (27/11/2026). Os gatilhos de db/migrations_sqlite/026_descriptive_metadata.sql usam INDEXED BY com idx_listens_profile_track, idx_occurrence_credits_artist, idx_occurrences_release e o índice automático sqlite_autoindex_occurrence_credits_1. Sem um desses índices, o comando que dispara o gatilho falha («no such index»), inclusive inserir e excluir escutas. As ISSUE-195 (consolidação das migrações) e ISSUE-210 (fundir tracks em occurrences) vão recriar essas tabelas. Garantir que a consolidação e a fusão preservem ou substituam os índices e os gatilhos, com teste que falhe se algum nome deixar de existir; avaliar se INDEXED BY continua necessário, repetindo as medições da ISSUE-218.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Teste falha se um índice usado pelos gatilhos deixar de existir ou mudar de nome
- [ ] #2 ISSUE-195 e ISSUE-210 preservam ou substituem índices e gatilhos, com as medições repetidas
<!-- AC:END -->
