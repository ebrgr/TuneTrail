---
id: ISSUE-219
title: 219 — Capturar tags, ano e data enviados pelo cliente na escuta
status: Aberta
assignee: []
created_date: '2026-11-24T19:53:00Z'
updated_date: '2026-11-24T19:54:00Z'
labels:
  - ingestao
  - catalogo
milestone: m-13
dependencies:
  - ISSUE-214
references:
  - backlog/evidencias/issue-055/ESTUDO-COMPLEMENTAR.md
documentation:
  - doc-048
priority: low
type: feature
ordinal: 232000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 24/11/2026 (subsistema de enriquecimento compartilhado, começando pelas ISSUE-174 e 175) e o fluxo que ele descreveu: banco já salvo, Navidrome, MusicBrainz, depois ListenBrainz e os serviços comerciais. Hoje ListenBrainzSnapshot não copia additional_info.tags, embora o tipo as aceite, e o tipo não tem ano nem data. O adaptador do Navidrome 0.64.2 não envia esses campos, então o ganho para esse cliente é nulo; outros clientes podem enviar (confirmar com o Pano Scrobbler, ISSUE-191, e com o Jellyfin). ATENÇÃO: mudar o snapshot muda o digest da escuta, como a ISSUE-175 mostrou com o digest v3: exige compatibilidade com os formatos v1, v2 e v3, reenvio comparado pela projeção original e reimportação v2. Não altera o esquema, por isso não precisa entrar antes da ISSUE-195. Guardar como afirmação do cliente, com a mesma procedência por entidade.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Tags, ano e data enviados pelo cliente chegam a metadata_values com fonte 'cliente' e sem alterar escutas já aceitas
- [ ] #2 Digest compatível com v1, v2 e v3, com teste de reenvio e de reimportação v2
- [ ] #3 Clientes que enviam esses campos identificados e documentados
<!-- AC:END -->
