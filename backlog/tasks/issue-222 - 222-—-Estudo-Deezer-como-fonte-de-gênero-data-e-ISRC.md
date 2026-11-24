---
id: ISSUE-222
title: '222 — Estudo: Deezer como fonte de gênero, data e ISRC'
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
ordinal: 235000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 24/11/2026 (subsistema de enriquecimento compartilhado, começando pelas ISSUE-174 e 175) e o fluxo que ele descreveu: banco já salvo, Navidrome, MusicBrainz, depois ListenBrainz e os serviços comerciais. Não há verificação do que a API pública do Deezer devolve: não sei confirmar os campos de gênero, data de lançamento e ISRC da faixa. Verificar com chamadas reais a dados públicos: campos, limites de requisição, necessidade de chave, e os termos que restringem guardar e redistribuir os dados. Avaliar se o ISRC serve de ponte para o MusicBrainz (issue C). Não confundir com o importador de histórico do Deezer da ISSUE-119. Decidir se entra como fonte da cadeia B, sempre desligada por padrão.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Contrato verificado com respostas reais: campos de gênero, data e ISRC, e comportamento sem dado
- [ ] #2 Limites, necessidade de chave e termos de armazenamento registrados
- [ ] #3 Decisão adotar ou não adotar, e issue do provedor aberta se adotar
<!-- AC:END -->
