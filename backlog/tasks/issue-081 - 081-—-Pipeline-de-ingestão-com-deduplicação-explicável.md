---
id: ISSUE-081
title: 081 — Pipeline de ingestão com deduplicação explicável
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-080
priority: high
type: feature
ordinal: 81000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §4.2. `internal/ingest` (normalizar → deduplicar → gravar → enfileirar) e `internal/dedup`. Deduplicação dentro do perfil; casos duvidosos vão para revisão, nunca apagados automaticamente. Revisão obrigatória do Advisor antes de implementar (AGENTS.md: mensageria/concorrência/criptografia → @Opus effort=high no Claude, @Sol effort=medium no Codex).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 ListenBrainz e importações passam pelo pipeline sem mudar o comportamento observável
- [ ] #2 Cada decisão grava motivo em ingest_events
- [ ] #3 Testes de concorrência e de repetições legítimas
- [ ] #4 Parecer do Advisor registrado
<!-- AC:END -->
