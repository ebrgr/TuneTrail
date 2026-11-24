---
id: ISSUE-218
title: '218 — Navidrome como provedor de metadados: gêneros, ano, datas, ISRC e BPM'
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
ordinal: 231000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 24/11/2026 (subsistema de enriquecimento compartilhado, começando pelas ISSUE-174 e 175) e o fluxo que ele descreveu: banco já salvo, Navidrome, MusicBrainz, depois ListenBrainz e os serviços comerciais. Cadeia B, etapa Navidrome. Amostra aleatória do servidor do usuário em 24/11/2026 (presença de campo, não qualidade nem censo): em 100 músicas, 78 tinham genres, 98 year, 41 musicBrainzId, 26 isrc e 14 bpm; em 15 álbuns, releaseDate e originalReleaseDate em 15, genres em 11, recordLabels em 11, musicBrainzId em 5 e releaseTypes em 5. O adaptador de escuta do Navidrome 0.64.2 não envia gêneros, tags, ano, datas, ISRC nem BPM, então só a consulta à API Subsonic traz esses dados. Usar o ID nativo do servidor apenas quando comprovadamente associado a ele; caso contrário, buscar candidatos por search3 e exigir correspondência inequívoca (já feito na ISSUE-174). Depois de uma associação segura, getSong e getAlbum fornecem os campos adicionais, guardando a origem por entidade em metadata_values. Confirmar a presença de originalReleaseDate na resposta de álbum, que não consta no tipo Child. O ISRC alimenta a ponte da issue C. Não registrar credenciais nem URLs autenticadas em evidências, logs ou testes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Gêneros, ano e datas do Navidrome gravados com fonte e instância, só depois de associação segura da faixa e do álbum
- [ ] #2 ISRC e BPM guardados quando existirem, e o ISRC disponível para a ponte com o MusicBrainz
- [ ] #3 Campos ausentes ou vazios não geram pendência nem apagam dado anterior
- [ ] #4 Testes com respostas simuladas, sem credenciais, cobrindo versões repetidas e resultados truncados
<!-- AC:END -->
