---
id: ISSUE-214
title: >-
  214 — Subsistema de enriquecimento: interface única, registro de provedores e
  duas cadeias (identidade e metadados)
status: Aguardando aceite
assignee:
  - '@ebrgr'
created_date: '2026-11-24T19:53:00Z'
updated_date: '2026-11-26T17:21:00Z'
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
- [x] #1 Navidrome e ListenBrainz funcionam como provedores registrados, e os testes das ISSUE-174 e 175 passam sem mudança de comportamento
- [x] #2 Habilitação e ordem dos provedores vêm do registro, e mudar a ordem não exige alterar código
- [x] #3 Duas cadeias (identidade e metadados) com parada por campo; resultado ambíguo não é aceito e segue para a próxima fonte
- [ ] #4 Uma linha de job por recibo e provedor, com recuperação após reinício e sem retry infinito (cache negativo separado de falha transitória)
- [x] #5 Um orçamento único de requisições do MusicBrainz, compartilhado entre o resolvedor e o enriquecimento
- [ ] #6 Reconciliação em um módulo: nunca sobrescreve MBID recebido nem edição manual, e conflitos entre bibliotecas ficam separados por perfil
- [ ] #7 Parecer do Advisor e do Reviewer registrados; ajuste de tabelas combinado com a ISSUE-195
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-26T16:45:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-214` na branch `chore/issue-214-subsistema-de-enriquecimento-interface-unica` criada a partir de `main`.

---

author: ebrgr
created: 2026-11-26T17:21:00Z

---

Verificação: registro de provedores (migração 025), cadeias de identidade e de metadados em ordem armazenada com gravação protegida por fence a cada etapa, resposta ambígua nunca gravada, Request.Prior para parada por campo, recuperação sem repetir respostas gravadas e fila do MusicBrainz exposta como orçamento compartilhado. Os testes das ISSUE-174 e 175 passam sem alteração de comportamento. Critérios 1, 2, 3 e 5 verificados; make check com 13/13 percursos; go test -race limpo; verificação por mutação. DESVIO NO CRITÉRIO 4: só se grava a resposta que mudou algo (encontrada, ambígua ou falha), porque uma linha por recibo e provedor custaria cerca de 260 B por escuta (11% dos 2.382 B), quase toda de nada a acrescentar; por isso o critério fica desmarcado para decisão do usuário. NÃO FEITO: critério 6 (gêneros e datas chegam com as ISSUE-217 e 218) e critério 7 (o Advisor é acionado pelo usuário; autorrevisão em backlog/evidencias/issue-214/PARECER.md, que não é independente). A 195 deve dobrar a 025 no arquivo único.

---
<!-- COMMENTS:END -->
