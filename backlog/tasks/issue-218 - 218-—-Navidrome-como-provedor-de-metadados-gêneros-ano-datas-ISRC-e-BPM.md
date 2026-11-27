---
id: ISSUE-218
title: '218 — Navidrome como provedor de metadados: gêneros, ano, datas, ISRC e BPM'
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
ordinal: 231000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 24/11/2026 (subsistema de enriquecimento compartilhado, começando pelas ISSUE-174 e 175) e o fluxo que ele descreveu: banco já salvo, Navidrome, MusicBrainz, depois ListenBrainz e os serviços comerciais. Cadeia B, etapa Navidrome. Amostra aleatória do servidor do usuário em 24/11/2026 (presença de campo, não qualidade nem censo): em 100 músicas, 78 tinham genres, 98 year, 41 musicBrainzId, 26 isrc e 14 bpm; em 15 álbuns, releaseDate e originalReleaseDate em 15, genres em 11, recordLabels em 11, musicBrainzId em 5 e releaseTypes em 5. O adaptador de escuta do Navidrome 0.64.2 não envia gêneros, tags, ano, datas, ISRC nem BPM, então só a consulta à API Subsonic traz esses dados. Usar o ID nativo do servidor apenas quando comprovadamente associado a ele; caso contrário, buscar candidatos por search3 e exigir correspondência inequívoca (já feito na ISSUE-174). Depois de uma associação segura, getSong e getAlbum fornecem os campos adicionais, guardando a origem por entidade em metadata_values. Confirmar a presença de originalReleaseDate na resposta de álbum, que não consta no tipo Child. O ISRC alimenta a ponte por ISRC da ISSUE-216. Não registrar credenciais nem URLs autenticadas em evidências, logs ou testes.

**Correção de 27/11/2026 (conferida no código da v0.79).** (a) `metadata_values` (migração 024) só aceita os tipos `mbid`, `genre`, `tag`, `recording_year`, `release_date` e `artist_origin_country`: ISRC e BPM exigem tipos novos e uma migração, que no SQLite recria a tabela. O texto anterior e o doc-048 diziam que a tabela não mudaria, e isso estava errado. Proposta do coordenador, ainda a confirmar pelo usuário: uma única migração cobrindo os tipos desta issue e da ISSUE-217, entregue antes da ISSUE-195. (b) Hoje só `mbid` é gravado (`writeMBIDDeclaration` em `internal/db/sqlite/enrichment.go` ignora os demais campos): a primeira entre a 217 e a 218 constrói a escrita dos outros tipos, com o vínculo à entidade certa; a leitura e a tela são da ISSUE-212. (c) A cadeia de identidade roda antes da de metadados. Um ISRC colhido na cadeia de metadados só chega à busca do MusicBrainz nas escutas seguintes, a menos que a etapa do Navidrome na cadeia de identidade também o colha. (d) A consulta por ISRC da ISSUE-216 (`lookupISRC`) existe, testada, e sem chamador: o critério 3 daquela issue ficou aberto à espera do ISRC.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Gêneros, ano e datas do Navidrome gravados com fonte e instância, só depois de associação segura da faixa e do álbum
- [ ] #2 ISRC e BPM guardados quando existirem, em tipos novos de metadata_values criados por migração, e o ISRC disponível para a ponte com o MusicBrainz
- [ ] #3 Campos ausentes ou vazios não geram pendência nem apagam dado anterior
- [ ] #4 Testes com respostas simuladas, sem credenciais, cobrindo versões repetidas e resultados truncados
- [ ] #5 Escrita em metadata_values dos tipos que não são mbid, com vínculo à entidade certa (se a ISSUE-217 não a tiver construído antes), e migração coordenada com a ISSUE-195
- [ ] #6 A consulta por ISRC da ISSUE-216 ligada à cadeia de identidade, usando o ISRC colhido do Navidrome, com ISRC que leva a mais de uma gravação tratado como ambíguo
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-27T15:00:00Z

---

27/11/2026: texto corrigido. ISRC e BPM não são aceitos por metadata_values (exigem migração), a referência «issue C» era um resíduo e passa a ser a ISSUE-216, e a ordem das cadeias atrasa o uso do ISRC. Critério 2 ajustado; acrescentados o de escrita e migração e o de ligar a consulta por ISRC, pendência herdada do critério 3 da ISSUE-216.

---
<!-- COMMENTS:END -->
