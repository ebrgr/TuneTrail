---
id: ISSUE-131
title: 131 — Aliases secundários e conflitos de MBID no resolvedor de faixas
status: Aberta
assignee: []
created_date: '2026-11-15T16:43:00Z'
updated_date: '2026-11-15T16:59:00Z'
labels:
  - catalogo
  - ingestao
milestone: m-11
dependencies: []
documentation:
  - doc-036
priority: medium
type: bug
ordinal: 144000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte 1 do estudo da ISSUE-116 (doc-036). No resolvedor atual, um alias secundário cria faixa nova em vez de achar a existente (Koito #369), e títulos iguais com MusicBrainz IDs diferentes se confundem (PR #315, #214). Pequeno e sem migração.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Alias secundário no mesmo contexto reutiliza o ID existente
- [ ] #2 Alias ambíguo produz decisão explícita, sem escolha silenciosa
- [ ] #3 MBIDs diferentes com títulos iguais nunca retornam o primeiro com o MBID do segundo; retorno e banco concordam
- [ ] #4 Testes cobrem artista adicional e ordem dos artistas
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-15T16:59:00Z

---

Reclassificada a pedido do usuário (2026-11-15): correção adiada, prioridade alta → média. Continua obrigatória antes da ISSUE-132, que já depende desta tarefa.

---
<!-- COMMENTS:END -->
