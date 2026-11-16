---
id: ISSUE-081
title: 081 — Pipeline de ingestão com deduplicação explicável
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-16T17:06:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-080
documentation:
  - doc-039
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

## Comments

<!-- COMMENTS:BEGIN -->
author: codex
created: 2026-11-16T17:06:00Z

---

Alinhamento após aceite da ISSUE-132: doc-039 e decision-008 exigem preservar recebimentos ambíguos/conflitantes e confirmar ListenBrainz após persistência durável, com revisão separada da deduplicação. O AC1 sobre comportamento observável deve preservar formato e compatibilidade dos clientes, incluindo esta mudança de semântica aprovada para conflitos; não exigir manter o HTTP 500 atual. Playing_now permanece efêmero.

---
<!-- COMMENTS:END -->
