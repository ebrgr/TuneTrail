---
id: ISSUE-216
title: >-
  216 — MusicBrainz como provedor de identidade: busca por nome e por ISRC com
  verificação
status: Aberta
assignee: []
created_date: '2026-11-24T19:53:00Z'
updated_date: '2026-11-24T19:54:00Z'
labels:
  - ingestao
  - catalogo
milestone: m-11
dependencies:
  - ISSUE-214
  - ISSUE-215
references:
  - backlog/evidencias/issue-055/ESTUDO-COMPLEMENTAR.md
documentation:
  - doc-048
priority: medium
type: feature
ordinal: 229000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 24/11/2026 (subsistema de enriquecimento compartilhado, começando pelas ISSUE-174 e 175) e o fluxo que ele descreveu: banco já salvo, Navidrome, MusicBrainz, depois ListenBrainz e os serviços comerciais. É a peça nova: hoje o MusicBrainz só é consultado por MBID que já veio na escuta (recording, release, release-group e apelidos do artista); não existe busca por nome. Cadeia A, etapa de identidade: buscar a gravação em /ws/2/recording?query= (sintaxe Lucene, com escape) e, quando o Navidrome fornecer o ISRC, consultar por ISRC. Todo candidato é verificado por duração, créditos de artista, título e álbum, com a regra e o limiar definidos no estudo da issue B; o score sozinho não basta. Ambíguo não é aceito e segue para a próxima fonte. O resultado é afirmação declarada, com fonte e data, nunca prova (contrato da ISSUE-134), e não sobrescreve MBID existente. Usa o cliente e a fila de 1 requisição por segundo já existentes, divididos com o resolvedor; em importações grandes tem prioridade baixa e teto por minuto; mantém o User-Agent identificado, o espelho próprio por MUSICBRAINZ_URL e a chave TUNETRAIL_DISABLE_MUSICBRAINZ. Roda fora do caminho do ACK e fora do lock de escrita.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Escuta sem MBID recebe o MBID de uma gravação verificada por duração, créditos e álbum, como declaração com fonte e data
- [ ] #2 Candidato ambíguo, divergente ou abaixo do limiar não é aceito e a cadeia segue; nenhum MBID existente é sobrescrito
- [ ] #3 Consulta por ISRC quando o Navidrome o fornece, tratando ISRC com mais de uma gravação como ambíguo
- [ ] #4 Fila compartilhada com o resolvedor, prioridade baixa em importações e funcionamento com o MusicBrainz desligado
- [ ] #5 Testes com respostas simuladas: versões ao vivo e de estúdio, título repetido, duração divergente, 503 e limite de taxa
<!-- AC:END -->
