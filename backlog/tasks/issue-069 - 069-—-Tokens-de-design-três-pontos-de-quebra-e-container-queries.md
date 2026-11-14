---
id: ISSUE-069
title: 069 — Tokens de design, três pontos de quebra e container queries
status: Aberta
assignee: []
created_date: '2026-11-14T00:38:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-068
priority: high
type: enhancement
ordinal: 69000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §3.1 e doc-031 fase 4. Muda o layout de propósito.

Passos:
1. Tokens: `--space-1..8`, escala tipográfica fluida `--step--1..5` com `clamp()`, raios e sombras.
2. Pontos de quebra únicos: 40rem, 64rem e 90rem (mobile-first, `min-width`); container queries em cards, heatmap e listas.
3. Eliminar larguras fixas em px de blocos (`.overview`, `.activity-card`, `.period-selector` e demais): `min()`, `clamp()`, grades `auto-fill`.
4. `check_web_style.py` rejeita `@media` fora da escala.
5. Capturas antes/depois e aceite visual do usuário.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Sem rolagem horizontal de página em 320, 360, 390, 768, 1024 e 1440px
- [ ] #2 Somente os três pontos de quebra aprovados no CSS (verificado no ui.check)
- [ ] #3 Nenhuma largura fixa em px para blocos de layout
- [ ] #4 Capturas antes/depois em Yuu e Pearl com aceite visual do usuário
<!-- AC:END -->
