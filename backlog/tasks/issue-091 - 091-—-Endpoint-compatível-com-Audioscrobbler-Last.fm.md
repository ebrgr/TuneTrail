---
id: ISSUE-091
title: 091 — Endpoint compatível com Audioscrobbler/Last.fm
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
labels:
  - api
  - ingestao
milestone: m-12
dependencies:
  - ISSUE-081
priority: low
type: feature
ordinal: 91000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §5.2 (prioridade 5). `/apis/audioscrobbler/2.0` mapeado ao mesmo pipeline, para players que só falam Last.fm. Revisão do Advisor recomendada (AGENTS.md: rotas/CRUD/configuração → @Opus effort=medium, @Sol effort=low).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Autenticação por chave de perfil
- [ ] #2 Testes com clientes reais documentados
<!-- AC:END -->
