---
id: ISSUE-079
title: 079 — Contrato de ingestão, retenção e revisão do Advisor
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-16T17:06:00Z'
labels:
  - ingestao
  - banco
milestone: m-11
dependencies:
  - ISSUE-064
references:
  - >-
    backlog/decisions/decision-008 -
    Identidade-musical-separada-em-gravação-ocorrência-e-evento.md
documentation:
  - doc-039
priority: high
type: spike
ordinal: 79000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §4. Revisão obrigatória do Advisor antes de implementar (AGENTS.md: mensageria/concorrência/criptografia → @Opus effort=high no Claude, @Sol effort=medium no Codex).

Definir: envelope de evento, decisões de deduplicação (Igual | Diferente | Revisar), retenção de `ingest_events` e guarda do payload bruto (decisão do usuário, doc-010 §10 item 3), e se a migração v4 inclui também as tabelas de metadados da ISSUE-055.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Contrato do envelope e da deduplicação documentado em backlog/docs/design
- [ ] #2 Decisão do usuário sobre retenção e payload registrada
- [ ] #3 Parecer do Advisor registrado na tarefa
- [ ] #4 Plano da migração v4 coordenado com a ISSUE-055
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: codex
created: 2026-11-16T17:06:00Z

---

Contrato da ISSUE-132 aceito em 16/11/2026 (doc-039, decision-008). Coordenar envelope de recebimentos, metadados mínimos de reprocessamento, revisão por perfil e recebimentos conflitantes. Retenção, prazo e guarda de payload bruto continuam decisões desta ISSUE-079; o aceite da 132 não os definiu.

---
<!-- COMMENTS:END -->
