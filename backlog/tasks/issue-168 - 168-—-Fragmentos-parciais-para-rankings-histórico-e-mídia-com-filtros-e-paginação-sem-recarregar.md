---
id: ISSUE-168
title: >-
  168 — Fragmentos parciais para rankings, histórico e mídia, com filtros e
  paginação sem recarregar
status: Aberta
assignee: []
created_date: '2026-11-19T00:22:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-075
priority: low
type: feature
ordinal: 181000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: recorte da ISSUE-075 em 2026-11-18, por decisão do usuário (caminho 2). A 075 ficou só com View Transitions, Speculation Rules e tooltips dos gráficos; esta tarefa recebe a parte de maior risco, que depende de JavaScript. Passos: (1) estender o ?partial=1 da ISSUE-107 (busca) aos rankings, ao histórico e às páginas de mídia; (2) o script intercepta formulários GET de filtros e paginação, troca só o <main> e mantém a URL com history.pushState; (3) preservar Voltar, foco do teclado e leitores de tela (anunciar a troca, mover o foco). Riscos a tratar: botão Voltar, foco e acessibilidade, e o orçamento de JavaScript de 25 KB comprimido no total. Melhoria progressiva sobre links e formulários que já funcionam sem JavaScript. Baixa prioridade; avaliar após a ISSUE-078 e antes ou depois do 1.0.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Todas as jornadas continuam funcionando com JavaScript desligado
- [ ] #2 Voltar, foco do teclado e leitor de tela verificados após a troca do main
- [ ] #3 JS total abaixo de 25 KB comprimido
<!-- AC:END -->
