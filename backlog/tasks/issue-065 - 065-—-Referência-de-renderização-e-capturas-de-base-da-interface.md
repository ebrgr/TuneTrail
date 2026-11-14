---
id: ISSUE-065
title: 065 — Referência de renderização e capturas de base da interface
status: Aberta
assignee: []
created_date: '2026-11-14T00:38:00Z'
labels:
  - ui
  - validacao
milestone: m-10
dependencies:
  - ISSUE-061
priority: high
type: chore
ordinal: 65000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-031 §Como verificar, passo 1. Pré-requisito de todas as tarefas da fase 1.

Passos:
1. Teste Go que renderiza todas as páginas da prévia, das páginas reais e da administração com dados fictícios e grava o HTML e o CSS servidos como referência.
2. Script de comparação: idêntico byte a byte e idêntico após normalizar espaços.
3. Capturas de base em 320, 360, 390, 768, 1024 e 1440px, temas Yuu e Pearl, com os scripts de captura atuais.
4. Guardar em `backlog/evidencias/ui-responsiva/base/`.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 HTML e CSS de referência gerados por comando repetível
- [ ] #2 Comparação byte a byte e normalizada disponível para as próximas tarefas
- [ ] #3 Capturas de base nas seis larguras e dois temas registradas
<!-- AC:END -->
