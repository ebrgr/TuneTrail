---
id: ISSUE-217
title: '217 — MusicBrainz como provedor de metadados por MBID: gêneros, tags e datas'
status: Aberta
assignee: []
created_date: '2026-11-24T19:53:00Z'
updated_date: '2026-11-24T19:54:00Z'
labels:
  - catalogo
  - ingestao
milestone: m-13
dependencies:
  - ISSUE-214
references:
  - backlog/evidencias/issue-055/ESTUDO-COMPLEMENTAR.md
documentation:
  - doc-048
priority: medium
type: feature
ordinal: 230000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 24/11/2026 (subsistema de enriquecimento compartilhado, começando pelas ISSUE-174 e 175) e o fluxo que ele descreveu: banco já salvo, Navidrome, MusicBrainz, depois ListenBrainz e os serviços comerciais. Cadeia B, etapa MusicBrainz: com o MBID da gravação, do lançamento ou do artista, consultar inc=genres+tags (gêneros distintos das tags gerais) e as datas. Preservar a precisão parcial (YYYY, YYYY-MM, YYYY-MM-DD) e distinguir a data da edição, o primeiro lançamento do grupo e a primeira publicação da gravação; não herdar o gênero do artista como fato da gravação sem marcar a inferência. Gravação, lançamento e grupo são entidades diferentes, e a API pode limitar as entidades associadas devolvidas. Os dados suplementares (tags) têm licença diferente da dos dados centrais: verificar antes de guardar e exibir. Grava em metadata_values (genre, tag, recording_year, release_date, artist_origin_country) com fonte, instância e data, por perfil, incluindo artistas (decisão de 24/11/2026). Ausência ou erro nunca apaga dado anterior. Usa a fila compartilhada de 1 requisição por segundo.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Gêneros e tags de gravação, lançamento e artista gravados em metadata_values, com tag popular separada de gênero
- [ ] #2 Datas com precisão parcial preservada e distinção entre edição, primeiro lançamento do grupo e primeira publicação da gravação
- [ ] #3 Ausência, erro ou resposta parcial não apagam dado anterior; edição manual prevalece
- [ ] #4 Licença dos dados suplementares verificada e registrada antes de exibir
- [ ] #5 Testes com respostas simuladas: dado parcial, lançamentos truncados, MBID inexistente e indisponibilidade
<!-- AC:END -->
