---
id: ISSUE-187
title: >-
  187 — Acompanhamentos da 186: relink de créditos sem artista e receipts
  release_missing antigos
status: Aberta
assignee: []
created_date: '2026-11-22T19:28:00Z'
labels:
  - ingestao
  - catalogo
milestone: m-11
dependencies:
  - ISSUE-186
type: feature
ordinal: 200000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: parecer do Advisor da ISSUE-186 (backlog/evidencias/issue-186/PARECER-ADVISOR.md), para avaliação posterior; nada aqui bloqueia a 186. (1) Créditos de ocorrência sem lançamento ficam com artist_id nulo quando o alias é ambíguo ou desconhecido e nunca são ligados depois: avaliar uma rotina de relink (quando o alias passar a identificar um único artista, ou por ação explícita de revisão), sem fusão por similaridade. (2) Recibos antigos em pending_identity/release_missing não são reenfileirados após a 186: tratar no corte da ISSUE-172 (reenfileirar para a nova regra ou descartar pela ação da ISSUE-185, com auditoria); medir quantos existem em bancos reais antes de decidir. (3) O ranking divide a mesma faixa entre a gravação do álbum e a gravação sem álbum sem MBID: aceito pelo contrato da ISSUE-132 (doc-039); reavaliar se a divisão incomodar na prática, por exemplo com sugestão de fusão explícita na revisão. (4) MBID de gravação declarado pela fonte cria a gravação com proveniência não validada: avaliar marcar a origem como declarada e validar depois pelo MusicBrainz. Item 2 deve ser decidido antes ou junto da ISSUE-172 fase 2.
<!-- SECTION:DESCRIPTION:END -->
