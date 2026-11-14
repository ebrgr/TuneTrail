---
id: ISSUE-090
title: 090 — Spotify por perfil com OAuth
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
labels:
  - ingestao
milestone: m-12
dependencies:
  - ISSUE-086
  - ISSUE-085
priority: low
type: feature
ordinal: 90000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ROADMAP item 4 e doc-010 §5.2 (prioridade 4). Requer decisão do usuário: Spotify nativo ou somente coletor externo (doc-010 §10 item 4). `recently-played` + `currently-playing`, app de desenvolvedor por usuário, tokens fora do JSON. Revisão obrigatória do Advisor antes de implementar (AGENTS.md: mensageria/concorrência/criptografia → @Opus effort=high no Claude, @Sol effort=medium no Codex).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Decisão do usuário registrada
- [ ] #2 OAuth com tokens protegidos e revogáveis
- [ ] #3 Polling sem perder faixas em teste com relógio injetável
<!-- AC:END -->
