---
id: ISSUE-236
title: '236 — Navidrome: decidir o que gravar quando só o álbum diverge'
status: Aberta
assignee: []
created_date: '2026-11-28T01:53:00Z'
updated_date: '2026-11-28T01:53:00Z'
labels:
  - ingestao
  - catalogo
milestone: m-13
dependencies:
  - ISSUE-218
type: task
ordinal: 249000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: parecer do Advisor da ISSUE-218 (27/11/2026). O critério 1 da ISSUE-218 pede gravar os valores do Navidrome só depois de associação segura da faixa e do álbum. A implementação entregue grava os valores da faixa (gênero, ano, ISRC e BPM) quando a faixa confere e apenas o álbum devolvido por getAlbum diverge (por exemplo, «Album (Deluxe)»), descartando os do álbum; a regra veio das instruções da primeira rodada de correções, sem decisão expressa do usuário. Decidir entre manter essa regra, anotando o critério, ou descartar também os valores da faixa nesse caso. Até a decisão, o comportamento entregue permanece.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Decisão do usuário registrada na tarefa
- [ ] #2 Comportamento, teste e critério da ISSUE-218 alinhados à decisão
<!-- AC:END -->
