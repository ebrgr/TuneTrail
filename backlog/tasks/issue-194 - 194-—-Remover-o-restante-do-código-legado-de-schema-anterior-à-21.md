---
id: ISSUE-194
title: 194 — Remover o restante do código legado de schema anterior à 21
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-23T00:10:00Z'
updated_date: '2026-11-23T01:29:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-172
references:
  - backlog/evidencias/issue-194/LIMPEZA.md
  - backlog/evidencias/issue-194/COLUNAS.md
type: chore
ordinal: 207000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pareceres do Reviewer e do Advisor da ISSUE-172 (backlog/evidencias/issue-172/PARECER-REVIEWER.md, PARECER-ADVISOR.md, LIMPEZA.md), decisão do usuário em 22/11/2026. A fase 2 da 172 removeu o escritor legado, a admissão em sombra, o despacho ao resolvedor v9, a ativação e o interruptor (−4.102 linhas), mas deixou código que só roda em schema anterior à 21 e é inalcançável em produção (v2 direto para a última). Escopo: (1) remover finishIngest e finalizeActiveReceipt e o que só eles usam, portando as fixtures históricas de internal/db/sqlite/historical_resolution_test.go para semear recibos por SQL e exercitar o resolvedor atual; (2) remover os ramos de compatibilidade e as consultas a profile_ingest_flags em ingestReviewEnabled, healthReviews e SetIngestReview (ingest_review.go), insertRelayIntent (relay_outbox.go), RelayPolicyProfiles (relay_panel.go), ingestSchemaVersion e demais chamadores; (3) o teste de referência da interface que ainda gera a página de saúde com fixture do schema 20; (4) instalar ou usar staticcheck e deadcode (com cache de módulos ou rede) para achar mais código morto e confirmar cada candidato; (5) manter a escuta ao vivo (PublishNowPlaying, ReadNowPlaying, rota playing_now) e os testes v2 para 21, v20 para 21 e de backup; migrações antigas não se editam. Sem mudança de comportamento em schema 21. Reviewer ao final; Advisor opcional.
<!-- SECTION:DESCRIPTION:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-23T00:31:00Z

---

Ordem registrada (22/11/2026): a ISSUE-195 (consolidar as migrações em um único salto da v2) depende desta e da ISSUE-055. Ao limpar, anotar quais colunas de transição de ingest_events (mode, resolver_variant, delivery_policy, parked_*, retry/claim bases) ficam sem leitor, para a 195 decidir se saem do schema final.

---

author: claude
created: 2026-11-23T00:40:00Z

---

Início (22/11/2026): despachada ao Codex em worktree própria. Primeira da fila do banco (194 → 055 → 190 → 193 → 195).

---

author: ebrgr
created: 2026-11-23T00:40:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-194` na branch `chore/issue-194-remover-o-restante-do-codigo` criada a partir de `main`.

---

author: codex
created: 2026-11-23T01:00:00Z

---

Verificação concluída. Removi os caminhos pré-21 e portei as fixtures de recibos para o resolvedor por ocorrência. As expectativas retiradas dependiam do escritor v9 ou das regras antigas de ativação e rotação de token, descritas em LIMPEZA.md. As migrações e o backup passaram, make check passou com 13/13 E2E e go test e race passaram. O binário final confirmou playing_now e single na porta 4127, com servidor encerrado. Ensaio isolado com 109 escutas aceitas, paridade de digest e ACK p95 de 5,83 ms. Inventário de colunas registrado para a ISSUE-195. Tarefa segue Em andamento.

---

author: claude
created: 2026-11-23T01:20:00Z

---

Reviewer: APROVADO, sem BUG. make check 13/13; -race ok em sqlite, importer, ingest, catalog, relay, admin; sem referências remanescentes em produção; escuta ao vivo e carga conferidas.

---

author: ebrgr
created: 2026-11-23T01:27:00Z

---

Verificação: Reviewer APROVADO; make check 13/13; -race ok; escuta ao vivo e carga conferidas

---

author: evertonberger
created: 2026-11-23T01:27:00Z

---

Aceite do usuário em 22/11/2026 ('aceito a 194').

---

author: ebrgr
created: 2026-11-23T01:29:00Z

---

Publicação: versão 0.67 (tag `v0.67`), branch `chore/issue-194-remover-o-restante-do-codigo` integrada ao `main` por fast-forward.

---
<!-- COMMENTS:END -->
