---
id: ISSUE-078
title: 078 — Auditoria de acessibilidade, desempenho e aceite da UI responsiva
status: Aberta
assignee: []
created_date: '2026-11-14T00:38:00Z'
labels:
  - ui
  - validacao
milestone: m-10
dependencies:
  - ISSUE-072
  - ISSUE-073
  - ISSUE-074
  - ISSUE-075
  - ISSUE-070
priority: high
type: chore
ordinal: 78000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §3.6 e doc-017 (ISSUE-024/026). Fecha a fase 1.

Passos:
1. WCAG 2.2 AA: foco visível, teclado, erros de formulário, tabelas, modal, leitor de tela, zoom de 200% e contraste de todos os temas.
2. Desempenho: LCP < 1,5s em 4G simulada com 10 mil escutas; CSS < 40 KB e JS < 25 KB comprimidos.
3. Capturas finais nas seis larguras, Yuu e Pearl, comparadas com a base.
4. Registrar evidências e pedir aceite do usuário.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Relatório de acessibilidade sem falhas bloqueantes
- [ ] #2 Metas de desempenho e tamanho medidas e cumpridas
- [ ] #3 Capturas finais comparadas com a base
- [ ] #4 Aceite do usuário para a fase 1
<!-- AC:END -->
