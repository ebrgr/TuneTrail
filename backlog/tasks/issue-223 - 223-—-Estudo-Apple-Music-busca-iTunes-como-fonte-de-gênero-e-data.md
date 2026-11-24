---
id: ISSUE-223
title: '223 — Estudo: Apple Music (busca iTunes) como fonte de gênero e data'
status: Aberta
assignee: []
created_date: '2026-11-24T19:53:00Z'
updated_date: '2026-11-24T19:54:00Z'
labels:
  - ingestao
  - docs
milestone: m-13
dependencies: []
references:
  - backlog/evidencias/issue-055/ESTUDO-COMPLEMENTAR.md
documentation:
  - doc-048
priority: low
type: spike
ordinal: 236000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 24/11/2026 (subsistema de enriquecimento compartilhado, começando pelas ISSUE-174 e 175) e o fluxo que ele descreveu: banco já salvo, Navidrome, MusicBrainz, depois ListenBrainz e os serviços comerciais. A busca do iTunes devolve, segundo a documentação da Apple, primaryGenreName, releaseDate e trackTimeMillis; não confirmei se devolve ISRC nem MBID, e a duração pode faltar em alguns tipos de mídia. A Apple orienta guardar em cache as consultas. Verificar com chamadas reais a dados públicos: campos, limites de requisição e os termos que restringem guardar e redistribuir os dados. Decidir se entra como fonte da cadeia B (gênero principal e data), sempre desligada por padrão. Não confundir com o importador de histórico do Apple Music da ISSUE-119.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Contrato verificado com respostas reais: gênero principal, data, duração e eventual ISRC
- [ ] #2 Limites, política de cache e termos de armazenamento registrados
- [ ] #3 Decisão adotar ou não adotar, e issue do provedor aberta se adotar
<!-- AC:END -->
