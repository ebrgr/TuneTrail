---
id: ISSUE-190
title: >-
  190 — Tempo realmente ouvido: duration_played do ListenBrainz e das
  importações
status: Aberta
assignee: []
created_date: '2026-11-22T20:05:00Z'
updated_date: '2026-11-23T00:38:00Z'
labels:
  - ingestao
  - catalogo
milestone: m-11
dependencies:
  - ISSUE-172
type: feature
ordinal: 203000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: PR gabehf/Koito#379 (fecha o #354; aberto, não mesclado), revisado em 22/11/2026. Hoje o tempo ouvido usa a duração da faixa, mesmo quando só metade foi tocada. Aceitar duration_played (segundos) e duration_played_ms (convertido) do additional_info do ListenBrainz, guardar por escuta e usar COALESCE(duration_played, duração da faixa) nas contagens de tempo ouvido (faixa, álbum, artista, período, Rewind). Casos de teste do PR a reaproveitar: 120 s informado, 90,5 s via _ms convertido, ausente usa a duração da faixa (soma 120+90+200). No TuneTrail: campo no Snapshot (ingest), coluna por escuta com NULL para desconhecido (o PR usa 0, evitar), limite para não passar da duração da faixa, inclusão na exportação/reimportação (ISSUE-183), no relay e na importação do ListenBrainz e do Spotify (ms_played). Não copiar o código do PR: depende do banco e das consultas antigas. Toca o Snapshot e as consultas de tempo ouvido: fazer DEPOIS da ISSUE-172 fase 2, para não conflitar com a remoção do escritor legado. Advisor opcional; re-medir as consultas de tempo ouvido.
<!-- SECTION:DESCRIPTION:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-23T00:38:00Z

---

Decisão do usuário (22/11/2026): esta issue está no mínimo que altera o banco antes do primeiro upgrade de produção e antecede a consolidação das migrações (ISSUE-195). Migração aditiva agora; depois dobrada no arquivo único da 195.

---
<!-- COMMENTS:END -->
