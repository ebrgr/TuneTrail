---
id: ISSUE-089
title: 089 — Webhooks do Jellyfin e do Plex
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
labels:
  - ingestao
  - api
milestone: m-12
dependencies:
  - ISSUE-081
priority: medium
type: feature
ordinal: 89000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §5.2 (prioridade 3). Rota de webhook por perfil com segredo, validação de assinatura quando houver e limites de corpo. Revisão do Advisor recomendada (AGENTS.md: rotas/CRUD/configuração → @Opus effort=medium, @Sol effort=low).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Webhooks autenticados por perfil
- [ ] #2 Eventos chegam pelo pipeline com testes de payload real
<!-- AC:END -->
