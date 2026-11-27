---
id: ISSUE-217
title: '217 — MusicBrainz como provedor de metadados por MBID: gêneros, tags e datas'
status: Aberta
assignee: []
created_date: '2026-11-24T19:53:00Z'
updated_date: '2026-11-27T15:00:00Z'
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

**Correção de 27/11/2026 (conferida no código da v0.79).** (a) O esquema não guarda as três datas: o catálogo não tem entidade de grupo de lançamento nem colunas de data em `releases`, `recordings` ou `artists`; em `metadata_values`, `release_date` se liga à edição e `recording_year` guarda só o ano da gravação. A data do primeiro lançamento do grupo e a data completa da primeira publicação da gravação exigem tipos novos e uma migração. Proposta do coordenador, ainda a confirmar pelo usuário: uma única migração com os tipos desta issue e da ISSUE-218, entregue antes da ISSUE-195. (b) Hoje só `mbid` é gravado (`writeMBIDDeclaration` em `internal/db/sqlite/enrichment.go`): a primeira entre a 217 e a 218 constrói a escrita dos demais tipos; a leitura e a tela são da ISSUE-212. (c) Não existe hoje edição manual de gênero, tag ou data: «edição manual prevalece» vale como regra de gravação, para quando ela existir. (d) A etapa de metadados precisa dos MBIDs que outras fontes gravaram: eles chegam pelo Snapshot com o catálogo e por `Request.Prior`.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Gêneros e tags de gravação, lançamento e artista gravados em metadata_values, com tag popular separada de gênero
- [ ] #2 Datas com precisão parcial preservada e distinção entre edição, primeiro lançamento do grupo e primeira publicação da gravação, em tipos de metadata_values criados por migração
- [ ] #3 Ausência, erro ou resposta parcial não apagam dado anterior, e a gravação nunca sobrescreve um valor de origem manual quando ela existir
- [ ] #4 Licença dos dados suplementares verificada e registrada antes de exibir
- [ ] #5 Testes com respostas simuladas: dado parcial, lançamentos truncados, MBID inexistente e indisponibilidade
- [ ] #6 Escrita em metadata_values dos tipos que não são mbid, com vínculo à entidade certa (se a ISSUE-218 não a tiver construído antes), e migração coordenada com a ISSUE-195
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-27T15:00:00Z

---

27/11/2026: texto corrigido. As três datas pedidas no critério 2 não cabem no esquema atual (exigem tipos novos e migração), só mbid é gravado hoje e não há edição manual de gênero, tag ou data. Critérios 2 e 3 ajustados e um critério de escrita e migração acrescentado.

---
<!-- COMMENTS:END -->
