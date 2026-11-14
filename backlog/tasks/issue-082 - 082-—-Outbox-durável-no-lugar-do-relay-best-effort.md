---
id: ISSUE-082
title: 082 — Outbox durável no lugar do relay best-effort
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
labels:
  - ingestao
  - api
milestone: m-11
dependencies:
  - ISSUE-081
priority: high
type: feature
ordinal: 82000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §4.2 e doc-017 (ISSUE-020/034). Worker com backoff exponencial e jitter, `Retry-After`, repetição de 429 e drenagem no encerramento; chave de idempotência única. Revisão obrigatória do Advisor antes de implementar (AGENTS.md: mensageria/concorrência/criptografia → @Opus effort=high no Claude, @Sol effort=medium no Codex).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Relay vira destino da outbox preservando o comportamento atual
- [ ] #2 Falhas e 429 são repetidas com backoff e Retry-After
- [ ] #3 Encerramento drena ou persiste a fila sem perda
- [ ] #4 Parecer do Advisor registrado
<!-- AC:END -->
