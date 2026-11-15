---
id: ISSUE-141
title: 141 — Escuta individual no histórico com página e diálogo
status: Aberta
assignee: []
created_date: '2026-11-15T20:04:00Z'
updated_date: '2026-11-15T20:13:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-139
  - ISSUE-140
references:
  - backlog/evidencias/issue-108/avaliacao-codex.md
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
- [ ] #1 Mesma escuta na página e no diálogo, identificada por perfil, faixa e instante; exclusão não altera escuta de outro perfil
- [ ] #2 Linha, contagens e estado vazio coerentes após excluir; rolagem e foco preservados; sem JavaScript volta ao contexto de origem, inclusive página de mídia
- [ ] #3 Visitante não recebe o link de gestão; × rápido continua funcionando
- [ ] #4 Testes de dois perfis com a mesma faixa e instante, retornos cruzados e captura do diálogo
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-15T20:13:00Z

---

Detalhes da escuta aprovados pelo usuário em 15/11/2026: faixa, artistas e álbum com links, data e hora no fuso do perfil, cliente que registrou (campo client) e exclusão com confirmação. O × rápido do histórico continua. Duração ouvida não entra: o banco não guarda esse dado.

---
<!-- COMMENTS:END -->
