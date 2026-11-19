---
id: ISSUE-170
title: 170 — Corrigir o MBID de artista de mbid_mapping descartado no submit-listens
status: Aberta
assignee: []
created_date: '2026-11-19T16:15:00Z'
labels:
  - api
milestone: m-11
dependencies: []
priority: medium
type: bug
ordinal: 183000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: estudo da ISSUE-081 (doc-043, 2026-11-19). Em engine/handlers/lbz_submit_listen.go:203 o resultado de ParseUUIDSlice(mbid_mapping.ArtistMBIDs) é descartado, então o MBID do artista vindo de mbid_mapping nunca é usado na resolução da escuta. Corrigir o uso do resultado (e o tratamento de erro), com testes de compatibilidade com payloads ListenBrainz reais e verificação do efeito na resolução de artista (MBID, depois alias, depois nome) sem criar artistas duplicados. Tarefa pequena, separada da ISSUE-081; coordenar com a 081a só se tocar o mesmo trecho.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 O MBID de artista de mbid_mapping é usado na resolução da escuta, com teste de regressão
- [ ] #2 Payloads ListenBrainz existentes continuam aceitos e sem duplicar artistas
<!-- AC:END -->
