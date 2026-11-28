---
id: ISSUE-239
title: 239 — Não repetir a identificação por nome ou ISRC que não teve resultado
status: Aberta
assignee: []
created_date: '2026-11-28T02:22:00Z'
updated_date: '2026-11-28T02:22:00Z'
labels:
  - ingestao
  - catalogo
milestone: m-13
dependencies:
  - ISSUE-217
type: enhancement
ordinal: 252000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: limite da ISSUE-217 informado ao usuário em 27/11/2026. A regra de não consultar de novo o que não teve resultado vale para os metadados do Navidrome e do MusicBrainz, mas não para a busca de identidade do MusicBrainz por nome ou ISRC em escutas sem MBID, que se repete a cada escuta passado o cache em memória. Decidir se a regra deve valer também para a identidade (e para os demais provedores de identidade) e, se valer, registrar a resposta definitiva sem resultado reaproveitando a estrutura da migração 027, com o mesmo acesso pela central de saúde. Falha nunca registra nada; mudança nos dados da escuta ou do catálogo libera a consulta.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Decisão do usuário sobre estender a regra à identidade registrada na tarefa
- [ ] #2 Se aprovada: escutas repetidas sem resultado não geram chamadas, falha continua sendo tentada e mudança nos dados libera a consulta, com custo em bytes medido antes
<!-- AC:END -->
