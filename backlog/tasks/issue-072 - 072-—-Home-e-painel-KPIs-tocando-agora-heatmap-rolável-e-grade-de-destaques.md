---
id: ISSUE-072
title: '072 — Home e painel: KPIs, tocando agora, heatmap rolável e grade de destaques'
status: Aberta
assignee: []
created_date: '2026-11-14T00:38:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-071
priority: medium
type: feature
ordinal: 72000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §3.4 (Home).

Passos:
1. Linha de KPIs em cards (minutos, escutas, artistas, sequência) com variação contra o período anterior.
2. "Tocando agora" no topo, quando houver.
3. Heatmap que rola horizontalmente em telas pequenas (`overflow-x`, `scroll-snap`), ancorado na semana atual.
4. Top artistas/álbuns em grade `auto-fill`: 2 colunas no celular, 4–6 no desktop.
5. Seletor de período sticky.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 KPIs com variação calculada no servidor e testada
- [ ] #2 Heatmap legível em 320px sem esconder semanas
- [ ] #3 Grade de destaques adaptável sem largura fixa
- [ ] #4 Capturas e aceite visual
<!-- AC:END -->
