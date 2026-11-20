---
id: ISSUE-082
title: 082 — Outbox durável no lugar do relay best-effort
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-20T22:08:00Z'
labels:
  - ingestao
  - api
milestone: m-11
dependencies:
  - ISSUE-081
references:
  - backlog/evidencias/issue-082/PLANO.md
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

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-20T22:04:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-082` na branch `feat/issue-082-outbox-duravel-no-lugar-do` criada a partir de `main`.

---

author: Codex
created: 2026-11-20T22:08:00Z

---

Fase 1: plano da outbox durável documentado, com mapa da v0.47, contrato de elegibilidade e payload, retry coordenado com a 173, migração 012, corte D4 e verificação por AC. Aguardando parecer obrigatório do Advisor (Opus high) e decisões sobre payload ativo, tentativas/idade, histórico e retenção antes da implementação.

---
<!-- COMMENTS:END -->
