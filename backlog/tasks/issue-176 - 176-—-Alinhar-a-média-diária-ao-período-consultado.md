---
id: ISSUE-176
title: 176 — Alinhar a média diária ao período consultado
status: Aberta
assignee: []
created_date: '2026-11-20T22:56:00Z'
updated_date: '2026-11-20T22:56:00Z'
labels:
  - api
  - ui
milestone: m-13
dependencies:
  - ISSUE-092
references:
  - backlog/evidencias/issue-092/SEMANTICA.md
  - backlog/evidencias/issue-092/BENCHMARK.md
type: bug
ordinal: 189000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-092 (semântica aprovada pelo usuário em 20/11/2026). Problema medido na v0.47: para as mesmas 60 escutas de janeiro de 2023, GET /stats?year=2023&month=1 devolve 0,50 por dia (denominador mistura dias ativos do mês com os do histórico: 3 e 120) e GET /summary devolve 1,94 por dia. Semântica aprovada: (1) média principal = escutas ÷ dias civis do recorte, incluindo datas com zero; a média por dia ativo, se exibida, fica separada e calculada só com os dias ativos do próprio recorte; (2) período em andamento (calendário ou YTD): um relógio e um fuso IANA para a consulta toda, eventos limitados ao instante da consulta e a data de hoje conta no denominador; rótulo «até agora»; janela móvel e calendário permanecem distintos; (3) DECISÃO DO USUÁRIO: o fuso NÃO aparece na interface (é usado internamente, vindo do perfil ou do navegador, e não é mostrado nem rotulado ao usuário); (4) eventos aceitos: uma escuta por evento vivo do perfil; pendências e duplicatas não somam. Aplicar em /stats, /summary e nos KPIs da Home e do Rewind que usam a média; manter compatibilidade do JSON (campos atuais, sem remover). Testes: limites de mês e ano, zero escutas, perfil vazio, período em andamento, DST e fuso, compatibilidade do JSON, paridade entre /stats e /summary. Coordenar períodos parciais e exclusões com a ISSUE-120 e as comparações com a ISSUE-093.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 /stats e /summary devolvem a mesma média diária para o mesmo recorte, calculada sobre os dias civis do recorte (incluindo dias com zero)
- [ ] #2 Período em andamento limitado ao instante da consulta, com a data de hoje no denominador e rótulo «até agora»; o fuso não é exibido ao usuário
- [ ] #3 Compatibilidade do JSON preservada; testes de limites, zero, perfil vazio, fuso/DST e paridade entre endpoints
<!-- AC:END -->
