---
id: ISSUE-077
title: 077 — Curvas mais suaves no gráfico de escutas sem ultrapassar os pontos
status: Aberta
assignee: []
created_date: '2026-11-14T00:38:00Z'
labels:
  - ui
milestone: m-10
dependencies: []
priority: low
type: enhancement
ordinal: 77000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-017 (ISSUE-047) — melhoria visual pequena.

Passos: estudar interpolação monotônica (por exemplo Fritsch–Carlson) que preserve limites, picos e zeros; comparar com a atual; manter os testes contra overshoot.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Interpolação nova sem overshoot, com testes
- [ ] #2 Comparação visual antes/depois com picos e zeros
<!-- AC:END -->
