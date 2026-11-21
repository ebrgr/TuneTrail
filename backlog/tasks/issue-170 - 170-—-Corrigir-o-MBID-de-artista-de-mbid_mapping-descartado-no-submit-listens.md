---
id: ISSUE-170
title: 170 — Corrigir o MBID de artista de mbid_mapping descartado no submit-listens
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-19T16:15:00Z'
updated_date: '2026-11-21T16:21:00Z'
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
Origem: estudo da ISSUE-081 (doc-043, 2026-11-19). O resultado de ParseUUIDSlice(mbid_mapping.ArtistMBIDs) é descartado, então o MBID do artista vindo de mbid_mapping nunca é usado na resolução da escuta.

Causa confirmada em 2026-11-21 em dois pontos: (1) caminho legado, engine/handlers/lbz_submit_listen.go (~linha 266): a chamada utils.ParseUUIDSlice(payload.TrackMeta.MBIDMapping.ArtistMBIDs) não atribui o retorno, e o if err seguinte lê um err antigo; o importador (internal/importer/listenbrainz.go:167) já atribui corretamente. (2) caminho ativo, internal/db/sqlite/ingest_resolve.go ingestCatalogOptions: o snapshot guarda MappedArtistMBIDs (internal/ingest/listenbrainz.go:24), mas só s.ArtistMBIDs vira opts.ArtistMbzIDs; gravação e lançamento já usam o fallback do mapeamento, artista não.

Regra a aplicar nos dois: usar o MBID de additional_info e, só quando não houver nenhum válido, o de mbid_mapping. Verificar o efeito na resolução de artista (MBID, depois alias, depois nome) sem criar artistas duplicados, com testes de compatibilidade com payloads ListenBrainz.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 O MBID de artista de mbid_mapping é usado na resolução da escuta, com teste de regressão
- [x] #2 Payloads ListenBrainz existentes continuam aceitos e sem duplicar artistas
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Correção em engine/handlers/lbz_submit_listen.go (atribui o retorno de ParseUUIDSlice) e internal/db/sqlite/ingest_resolve.go (fallback para MappedArtistMBIDs só quando additional_info não tem ID válido). Testes: engine/ingestion_issue170_test.go (legado e ativo; sem a correção ambos criam artista duplicado) e internal/db/sqlite/ingest_resolve_artist_mbid_test.go (prioridade e IDs inválidos). Verificado: go test de engine, internal/db/sqlite, importer, ingest e catalog. Não verificado: payload real do ListenBrainz em produção e comportamento com MusicBrainz online (os testes usam MBID já existente no catálogo).
<!-- SECTION:NOTES:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-21T16:19:00Z

---

Início: branch `fix/issue-170-corrigir-o-mbid-de-artista` criada a partir de `main`.

---
<!-- COMMENTS:END -->
