---
id: ISSUE-214
title: >-
  214 — Subsistema de enriquecimento: interface única, registro de provedores e
  duas cadeias (identidade e metadados)
status: Aberta
assignee: []
created_date: '2026-11-24T19:53:00Z'
updated_date: '2026-11-24T19:54:00Z'
labels:
  - ingestao
  - banco
milestone: m-11
dependencies: []
references:
  - backlog/evidencias/issue-055/ESTUDO-COMPLEMENTAR.md
documentation:
  - doc-048
priority: high
type: chore
ordinal: 227000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 24/11/2026 (subsistema de enriquecimento compartilhado, começando pelas ISSUE-174 e 175) e o fluxo que ele descreveu: banco já salvo, Navidrome, MusicBrainz, depois ListenBrainz e os serviços comerciais. Estado hoje: o Navidrome implementa a interface Provider, mas o ListenBrainz entra como parâmetro especial (ResolveNextEnrichmentWithFallback(ctx, provider, *ListenBrainz), com o token por chamada); enrichment_jobs guarda uma linha por recibo; enrichment_configuration é um interruptor global; o MusicBrainz vive em outro caminho (resolvedor e catálogo) com a própria fila de 1 requisição por segundo. Um terceiro provedor viraria outro parâmetro especial. OBJETIVO: (1) interface Provider com identificador, capacidades (quais campos entrega e se busca por ID ou por nome), disponibilidade por perfil (configuração e credencial) e Lookup que devolve candidatos e um status (found, absent, ambiguous, disabled, failed). (2) Registro de provedores em tabela (ligado, prioridade, cadeia, limite, timeout) no lugar de enrichment_configuration. (3) Duas cadeias com a mesma ordem, que é configuração e não código: A, identidade (MBID): banco, Navidrome, MusicBrainz, ListenBrainz e comerciais; B, metadados (gênero, tag, ano, data): banco, Navidrome, MusicBrainz por MBID, ListenBrainz por MBID, comerciais. Para por campo, não por recibo; resultado ambíguo não é aceito e segue para a próxima fonte; se todas falharem o campo fica em branco, sem pendência. (4) Jobs por recibo e provedor. (5) Limitador por provedor, disjuntor e cache negativo (metadata_fetch_log ou equivalente), com a fila de 1 requisição por segundo do MusicBrainz compartilhada entre o resolvedor e o enriquecimento. (6) Reconciliação em um módulo: precedência por campo, nunca sobrescreve MBID recebido nem edição manual, valores conflitantes entre bibliotecas ficam separados por perfil. (7) Migrar o Navidrome (174) e o ListenBrainz (175) para provedores sem mudar o comportamento. ESQUEMA: metadata_values (024) já cobre os tipos mbid, genre, tag, recording_year, release_date e artist_origin_country por perfil e não muda; mudam enrichment_configuration e enrichment_jobs, que a produção ainda não viu, então o ajuste entra no arquivo único da ISSUE-195. A ISSUE-195 passa a depender desta issue, que carrega a migração que a decisão de 22/11/2026 pedia antes da consolidação. Pelo estudo da 055, o Advisor (effort high) revisa o contrato por tocar fila, concorrência e chamadas externas.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Navidrome e ListenBrainz funcionam como provedores registrados, e os testes das ISSUE-174 e 175 passam sem mudança de comportamento
- [ ] #2 Habilitação e ordem dos provedores vêm do registro, e mudar a ordem não exige alterar código
- [ ] #3 Duas cadeias (identidade e metadados) com parada por campo; resultado ambíguo não é aceito e segue para a próxima fonte
- [ ] #4 Uma linha de job por recibo e provedor, com recuperação após reinício e sem retry infinito (cache negativo separado de falha transitória)
- [ ] #5 Um orçamento único de requisições do MusicBrainz, compartilhado entre o resolvedor e o enriquecimento
- [ ] #6 Reconciliação em um módulo: nunca sobrescreve MBID recebido nem edição manual, e conflitos entre bibliotecas ficam separados por perfil
- [ ] #7 Parecer do Advisor e do Reviewer registrados; ajuste de tabelas combinado com a ISSUE-195
<!-- AC:END -->
