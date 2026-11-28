---
id: ISSUE-235
title: '235 — Navidrome: não repetir consultas de faixa que a biblioteca não tem'
status: Aberta
assignee: []
created_date: '2026-11-28T01:53:00Z'
updated_date: '2026-11-28T02:20:00Z'
labels:
  - ingestao
  - catalogo
milestone: m-13
dependencies:
  - ISSUE-218
type: enhancement
ordinal: 248000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: parecer do Advisor da ISSUE-218 (27/11/2026). A ISSUE-218 só deixa de consultar o Navidrome quando a faixa já tem metadado gravado daquela fonte e instância. Escuta com todos os IDs de uma faixa ausente do Navidrome, ou para a qual ele não devolve metadado de faixa, continua chamando search3 (e, havendo correspondência, getSong e getAlbum) a cada escuta, passado o cache em memória de 10 minutos. Na v0.81 essas escutas não geravam chamada. Proposta a estudar: marca negativa durável por perfil, ocorrência, fonte e instância, com prazo para nova tentativa, medindo o custo em bytes por escuta antes de implementar. Mudança de schema entra uma por vez (decision-011 A6) e precisa ser coordenada com a ISSUE-195.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Escutas repetidas de uma faixa sem metadado no Navidrome não geram chamadas dentro do prazo definido, medido por teste com servidor simulado
- [ ] #2 Falha passageira continua sendo tentada de novo
- [ ] #3 Custo em bytes por escuta medido antes da implementação e aprovado pelo usuário
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-28T02:20:00Z

---

27/11/2026: o escopo passa a ser atendido pela ISSUE-217 (marca durável por fonte, migração 027), por decisão do usuário sobre respostas sem dado. Permanece aberta até o aceite da 217; não implementar em separado.

---
<!-- COMMENTS:END -->
