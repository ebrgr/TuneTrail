---
id: ISSUE-076
title: 076 — App instalável com service worker mínimo e página offline
status: Aberta
assignee: []
created_date: '2026-11-14T00:38:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-068
  - ISSUE-070
priority: low
type: feature
ordinal: 76000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §3.5.

Passos:
1. Service worker que faz cache apenas dos assets com hash e de uma página offline.
2. Nunca guardar páginas com dados de perfil privados; respeitar logout.
3. Atualizar `site.webmanifest` com a identidade TuneTrail.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Instalável no Android e iOS (manifest válido)
- [ ] #2 Página offline exibida sem rede
- [ ] #3 Nenhum dado de perfil privado em cache
<!-- AC:END -->
