---
id: ISSUE-118
title: '118 — Estudo: favoritos e API de feedback do ListenBrainz'
status: Aberta
assignee: []
created_date: '2026-11-14T22:54:00Z'
updated_date: '2026-11-22T20:05:00Z'
labels:
  - api
milestone: m-12
dependencies: []
type: spike
ordinal: 127000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: estudo ISSUE-114 (doc-034). Favoritos/loved tracks (Koito #104, #312) e API de feedback do ListenBrainz (PR #328, #110), incluindo importação de favoritos do Last.fm/ListenBrainz e sincronização de favoritos do Jellyfin/Plex (comentário no roadmap Koito #284). Avaliar modelo por perfil, endpoints compatíveis e exibição. Relacionar com ISSUE-087 e ISSUE-089.
<!-- SECTION:DESCRIPTION:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-22T20:05:00Z

---

Do PR gabehf/Koito#382 (22/11/2026): referência de rotas de feedback do ListenBrainz: GET /feedback/user/{user_name}/get-feedback e POST /feedback/recording-feedback (tabela track_feedback com score), testado com o Pano Scrobbler. Não copiar o código (100% gerado por LLM, migração 003 conflita com a nossa numeração). Aqui {user_name} seria o perfil e o feedback seria por gravação (modelo de ocorrência/gravação, doc-039). Depende de escopo de leitura nas chaves (ISSUE-121).

---
<!-- COMMENTS:END -->
