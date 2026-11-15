---
id: ISSUE-134
title: 134 — Ingestão por ocorrência em todas as fontes
status: Aberta
assignee: []
created_date: '2026-11-15T16:43:00Z'
updated_date: '2026-11-15T16:44:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-133
  - ISSUE-081
documentation:
  - doc-036
type: feature
ordinal: 147000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte 4 do estudo da ISSUE-116 (doc-036). Ingestão grava a ocorrência ouvida em todas as fontes (ListenBrainz, Last.fm, importadores).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Mesmo MBID em A/B gera uma gravação e duas ocorrências
- [ ] #2 Duas faixas 'Intro' com IDs ou posições diferentes continuam separadas
- [ ] #3 Sem desambiguador, o evento é preservado para revisão
- [ ] #4 Reimportação idempotente e repetição legítima testadas conforme a ISSUE-081
<!-- AC:END -->
