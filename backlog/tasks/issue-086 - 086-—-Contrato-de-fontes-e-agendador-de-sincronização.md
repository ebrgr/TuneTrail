---
id: ISSUE-086
title: 086 — Contrato de fontes e agendador de sincronização
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
labels:
  - ingestao
milestone: m-12
dependencies:
  - ISSUE-082
priority: high
type: feature
ordinal: 86000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §5.1. Interface `Source` com capacidades, polling adaptativo, relógio injetável, `Retry-After`, checkpoint avançado só após gravação durável, parada por `context`. Revisão obrigatória do Advisor antes de implementar (AGENTS.md: mensageria/concorrência/criptografia → @Opus effort=high no Claude, @Sol effort=medium no Codex).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Contrato e agendador com testes de relógio injetável
- [ ] #2 Checkpoint nunca avança antes da gravação
- [ ] #3 Parecer do Advisor registrado
<!-- AC:END -->
