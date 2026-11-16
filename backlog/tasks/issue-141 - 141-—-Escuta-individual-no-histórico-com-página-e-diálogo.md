---
id: ISSUE-141
title: 141 — Escuta individual no histórico com página e diálogo
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-15T20:04:00Z'
updated_date: '2026-11-16T00:46:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-139
  - ISSUE-140
references:
  - backlog/evidencias/issue-108/avaliacao-codex.md
  - backlog/evidencias/issue-141/relatorio.md
priority: high
type: feature
ordinal: 154000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte C da ISSUE-108 (avaliação do Codex em backlog/evidencias/issue-108/avaliacao-codex.md, §5). O link Manage de cada escuta (home, /listens e páginas de mídia) passa a abrir a página da escuta (recorte A) em diálogo (recorte B). Detalhes da escuta, aprovados pelo usuário em 15/11/2026: faixa, artistas e álbum com links, data e hora no fuso do perfil e o cliente que registrou (campo client da escuta), mais a exclusão com confirmação. O × rápido do histórico continua como está. Exclusão remove só a escuta selecionada (perfil, faixa e instante) e atualiza linha, contagens e estado vazio sem perder rolagem e foco. Corrige o retorno de uma escuta aberta a partir da página de mídia, que o history_return atual parece rejeitar.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Mesma escuta na página e no diálogo, identificada por perfil, faixa e instante; exclusão não altera escuta de outro perfil
- [x] #2 Linha, contagens e estado vazio coerentes após excluir; rolagem e foco preservados; sem JavaScript volta ao contexto de origem, inclusive página de mídia
- [x] #3 Visitante não recebe o link de gestão; × rápido continua funcionando
- [x] #4 Testes de dois perfis com a mesma faixa e instante, retornos cruzados e captura do diálogo
- [x] #5 O × do histórico abre uma janela de confirmação com o aviso de que não existe desfazer; confirmar apaga só aquela escuta
- [x] #6 Com JavaScript o link Gerenciar não aparece separado; sem JavaScript continua disponível
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-15T20:13:00Z

---

Detalhes da escuta aprovados pelo usuário em 15/11/2026: faixa, artistas e álbum com links, data e hora no fuso do perfil, cliente que registrou (campo client) e exclusão com confirmação. O × rápido do histórico continua. Duração ouvida não entra: o banco não guarda esse dado.

---

author: claude
created: 2026-11-15T22:44:00Z

---

Ajuste pedido pelo usuário em 15/11/2026, ao revisar a ISSUE-140 em /evertonberger/track/37: (1) o × ao lado de cada escuta passa a abrir uma janela perguntando se realmente quer apagar e avisando que não existe desfazer; (2) o link Gerenciar abaixo da data e o × deixam de ser dois caminhos: a confirmação de apagar só aquela escuta acontece na janela aberta pelo ×, com os detalhes aprovados (faixa, artistas e álbum, data e hora, cliente). Sem JavaScript o link Gerenciar continua como alternativa. Substitui a decisão anterior de manter o × como está.

---

author: ebrgr
created: 2026-11-16T00:19:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-141` na branch `feat/issue-141-escuta-individual-no-historico-com` criada a partir de `main`.

---

author: codex
created: 2026-11-16T00:35:00Z

---

Verificação funcional: roteador principal com tickets obrigatórios, dois perfis na mesma faixa/instante e retornos cruzados; Chrome real nas cinco superfícies em Yuu/Pearl e desktop/celular, foco/Esc/rede/sessão expirada, atualização sem recarga e bfcache; capturas produzidas, sem aceite visual. Critério 3 segue o comentário 2: × com janela/confirmar substitui exclusão rápida. Revisão do diff reservada ao coordenador por decisão explícita do usuário; grafo da worktree será atualizado na main após publicação conforme orientação do coordenador.

---

author: ebrgr
created: 2026-11-16T00:36:00Z

---

Verificação: make check verde com 12/12 E2E, testes pelo roteador com tickets obrigatórios e isolamento de dois perfis, Chrome real em home/listens/faixa/álbum/artista (Yuu/Pearl, desktop/celular), confirmação, foco/Esc, rede, sessão expirada, fallback sem JS, contagens/estado vazio sem recarga e bfcache. Relatório/capturas em backlog/evidencias/issue-141/relatorio.md. Revisão do diff fica com o coordenador por decisão do usuário; aceite visual e aceite final pendentes.

---

author: claude
created: 2026-11-16T00:45:00Z

---

Aceite: aprovada pelo usuário em 15/11/2026 (pedido de publicação).

---

author: ebrgr
created: 2026-11-16T00:46:00Z

---

Publicação: versão 0.23 (tag `v0.23`), branch `feat/issue-141-escuta-individual-no-historico-com` integrada ao `main` por fast-forward.

---
<!-- COMMENTS:END -->
