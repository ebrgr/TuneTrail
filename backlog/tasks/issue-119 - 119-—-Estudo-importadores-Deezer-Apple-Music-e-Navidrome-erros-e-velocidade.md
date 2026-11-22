---
id: ISSUE-119
title: '119 — Estudo: importadores Deezer, Apple Music e Navidrome, erros e velocidade'
status: Aberta
assignee: []
created_date: '2026-11-14T22:54:00Z'
updated_date: '2026-11-22T20:05:00Z'
labels:
  - ingestao
milestone: m-11
dependencies: []
type: spike
ordinal: 128000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: estudo ISSUE-114 (doc-034). Importadores: Deezer (PR #376), Apple Music (PR #378), Navidrome (#375); discrepâncias e erros silenciosos do Spotify (#205, #221); Last.fm JSON com chaves (#210, verificar); track_mbid do multi-scrobbler (#208, verificar); artistas separados por vírgula com spotify_artist_ids (PR #323); velocidade (PR #335, #334 synchronous=NORMAL). Relacionar com ISSUE-079 e ISSUE-081. Saída: quais importadores adotar e contrato de erros.
<!-- SECTION:DESCRIPTION:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-22T20:05:00Z

---

Revisão dos PRs do Koito em 22/11/2026 (#376 Deezer, #378 Apple Music, #375 Navidrome), sem adotar o código. Formatos úteis: Deezer = .xlsx, aba 10_listeningHistory (faixa, artista, ISRC, álbum, IP, tempo em segundos com -1 = desconhecido, plataforma, modelo, data sem fuso), país em 2_customizationData; Apple Music = Play Activity CSV (200+ colunas), só eventos PLAY_END, Play Duration Milliseconds, timestamp ISO 8601, artista só pelo Library Tracks.json. Problemas dos PRs que o nosso importador deve evitar: data sem fuso lida como UTC (exige o fuso do perfil), MsPlayed que guarda segundos, linhas ruins ignoradas em silêncio (precisa de contrato de erros), busca do artista só por título em minúsculas e fallback no PRIMEIRO resultado do MusicBrainz (credita artista errado; aqui, sem artista confiável, a escuta fica pendente ou é rejeitada com motivo, como nas ISSUE-173/186), dependência nova excelize, código 95-100% gerado por LLM e acoplado ao catalog.SubmitListen do Koito. Importar pelo caminho de recibos/ocorrências (captureFileListen), com perfil explícito e deduplicação. Navidrome (#375): sem PR; avaliar leitura da tabela de scrobbles do Navidrome. Precisamos de amostras reais de exportação (o autor do #375 se ofereceu para fornecer). Item relacionado: duration_played nas importações na ISSUE-190.

---
<!-- COMMENTS:END -->
