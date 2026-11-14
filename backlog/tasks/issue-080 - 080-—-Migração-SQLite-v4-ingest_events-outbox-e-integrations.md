---
id: ISSUE-080
title: '080 — Migração SQLite v4: ingest_events, outbox e integrations'
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
labels:
  - banco
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-079
priority: high
type: feature
ordinal: 80000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §4.1. Migração aditiva a partir do v3, incluindo `tunetrail.db` e o legado `koito.db`; atualizar limites e validações do bootstrap.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Migração aditiva testada a partir de bancos v3 reais e legados
- [ ] #2 Bootstrap aceita v4 e recusa versões desconhecidas
- [ ] #3 Backup antes de migrar e ensaio de restauração
- [ ] #4 Nenhuma escuta descartada
<!-- AC:END -->
