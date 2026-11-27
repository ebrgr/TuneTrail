---
id: ISSUE-230
title: '230 — Arquitetura, passo 1: teste que proíbe imports entre camadas'
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-27T15:45:00Z'
updated_date: '2026-11-27T19:57:00Z'
labels:
  - validacao
  - processo
milestone: m-15
dependencies: []
priority: medium
type: chore
ordinal: 243000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pergunta do usuário em 27/11/2026 sobre a separação dos subsistemas e levantamento feito no código da v0.79 (linhas por pacote e grafo de imports). Criar um teste de arquitetura em Go, rodado pelo make check, que falhe quando um pacote importar outro que a regra proíbe, para a mistura não crescer. Regras propostas: (1) adminui, webui, adminweb e engine/handlers não importam internal/db/sqlite; (2) só o engine, que monta o programa, importa internal/db/sqlite; (3) nenhum SQL fora de internal/db/sqlite; (4) internal/ingest, internal/models e internal/db não importam adaptadores (sqlite, mbz) nem a interface web; (5) nenhum pacote de domínio importa adminui, webui ou engine; (6) internal/db/sqlite não ganha novos imports de pacotes de negócio. O teste passa hoje com uma lista explícita de exceções, cada uma com o motivo: sqlite importa catalog, enrichment, relay, mbz e dedup; relayoutbox importa sqlite. A lista só pode diminuir. O webui importar o adminui fica registrado para o estudo, sem proibição por enquanto.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Teste em Go que lê os imports de produção e falha em qualquer import proibido, rodado pelo make check
- [ ] #2 Lista de exceções explícita, com motivo por item, e o teste falha também quando uma exceção deixa de existir e continua listada
- [ ] #3 Regra de que não há SQL fora de internal/db/sqlite verificada pelo teste
- [ ] #4 Teste comprovado por mutação: um import proibido acrescentado de propósito faz o teste falhar
- [ ] #5 Regras e como acrescentar uma exceção documentadas
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-27T19:57:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-230` na branch `chore/issue-230-arquitetura-passo-1-teste-que` criada a partir de `main`.

---
<!-- COMMENTS:END -->
