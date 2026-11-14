---
id: ISSUE-084
title: 084 — Tocando agora em tempo real com Server-Sent Events
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
labels:
  - ui
  - api
milestone: m-11
dependencies:
  - ISSUE-081
  - ISSUE-072
priority: medium
type: feature
ordinal: 84000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §3.5, ISSUE-056. Broker por perfil, heartbeat de 25s, limite de conexões por IP, encerramento pelo `context`, pausa com `visibilitychange`. Revisão obrigatória do Advisor antes de implementar (AGENTS.md: mensageria/concorrência/criptografia → @Opus effort=high no Claude, @Sol effort=medium no Codex).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Atualização sem polling e sem vazamento entre perfis
- [ ] #2 Limites de conexão e encerramento testados
- [ ] #3 Página continua correta sem JavaScript
<!-- AC:END -->
