---
id: ISSUE-229
title: >-
  229 — Reenvio de ID externo confiável com revisão ligada vira duplicata
  silenciosa
status: Aberta
assignee: []
created_date: '2026-11-27T15:21:00Z'
updated_date: '2026-11-27T15:21:00Z'
labels:
  - ingestao
milestone: m-11
dependencies: []
priority: low
type: enhancement
ordinal: 242000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Decisão do usuário em 27/11/2026 (C6): o reenvio de um ID externo confiável com a revisão ligada deve ser tratado como duplicata silenciosa, sem abrir pendência. O comportamento atual (abrir pendência) vem do plano de tarefas e NÃO foi conferido no código: verificar primeiro.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Comportamento atual verificado e registrado
- [ ] #2 Reenvio de ID externo confiável resolvido como duplicata sem pendência, com teste com a revisão ligada
- [ ] #3 Duplicatas que não vêm de ID confiável continuam indo para revisão
<!-- AC:END -->
